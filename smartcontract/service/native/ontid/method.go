/*
 * Copyright (C) 2018 The DNA Authors
 * This file is part of The DNA library.
 *
 * The DNA is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The DNA is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with The DNA.  If not, see <http://www.gnu.org/licenses/>.
 */
package ontid

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/dnaproject2/DNA/account"
	"github.com/dnaproject2/DNA/common/log"
	"github.com/dnaproject2/DNA/common/serialization"
	"github.com/dnaproject2/DNA/core/states"
	"github.com/dnaproject2/DNA/core/types"
	"github.com/dnaproject2/DNA/smartcontract/service/native"
	"github.com/dnaproject2/DNA/smartcontract/service/native/utils"
	"github.com/ontio/ontology-crypto/keypair"
)

func regIdWithPublicKey(srvc *native.NativeService) ([]byte, error) {
	log.Debug("registerIdWithPublicKey")
	log.Debug("srvc.Input:", srvc.Input)
	// parse arguments
	args := bytes.NewBuffer(srvc.Input)
	// arg0: ID
	arg0, err := serialization.ReadVarBytes(args)
	if err != nil {
		return utils.BYTE_FALSE, errors.New("register ONT ID error: parsing argument 0 failed")
	} else if len(arg0) == 0 {
		return utils.BYTE_FALSE, errors.New("register ONT ID error: invalid length of argument 0")
	}
	log.Debug("arg 0:", hex.EncodeToString(arg0), string(arg0))
	// arg1: public key
	arg1, err := serialization.ReadVarBytes(args)
	if err != nil {
		return utils.BYTE_FALSE, errors.New("register ONT ID error: parsing argument 1 failed")
	}

	log.Debug("arg 1:", hex.EncodeToString(arg1))

	if len(arg0) == 0 || len(arg1) == 0 {
		return utils.BYTE_FALSE, errors.New("register ONT ID error: invalid argument")
	}

	if !account.VerifyID(string(arg0)) {
		return utils.BYTE_FALSE, errors.New("register ONT ID error: invalid ID")
	}

	key, err := encodeID(arg0)
	if err != nil {
		return utils.BYTE_FALSE, errors.New("register ONT ID error: " + err.Error())
	}

	if checkIDExistence(srvc, key) {
		return utils.BYTE_FALSE, errors.New("register ONT ID error: already registered")
	}

	public, err := keypair.DeserializePublicKey(arg1)
	if err != nil {
		log.Error(err)
		return utils.BYTE_FALSE, errors.New("register ONT ID error: invalid public key")
	}
	addr := types.AddressFromPubKey(public)
	if !srvc.ContextRef.CheckWitness(addr) {
		return utils.BYTE_FALSE, errors.New("register ONT ID error: checking witness failed")
	}

	// insert public key
	_, err = insertPk(srvc, key, arg1)
	if err != nil {
		return utils.BYTE_FALSE, errors.New("register ONT ID error: store public key error, " + err.Error())
	}
	// set flags
	srvc.CacheDB.Put(key, states.GenRawStorageItem([]byte{flag_exist}))

	triggerRegisterEvent(srvc, arg0)

	return utils.BYTE_TRUE, nil
}

func regIdWithAttributes(srvc *native.NativeService) ([]byte, error) {
	// parse arguments
	args := bytes.NewBuffer(srvc.Input)
	// arg0: ID
	arg0, err := serialization.ReadVarBytes(args)
	if err != nil {
		return utils.BYTE_FALSE, errors.New("register ID with attributes error: argument 0 error, " + err.Error())
	} else if len(arg0) == 0 {
		return utils.BYTE_FALSE, errors.New("register ID with attributes error: argument 0 error, invalid length")
	}
	if !account.VerifyID(string(arg0)) {
		return utils.BYTE_FALSE, errors.New("register ONT ID error: invalid ID")
	}

	// arg1: public key
	arg1, err := serialization.ReadVarBytes(args)
	if err != nil {
		return utils.BYTE_FALSE, errors.New("register ID with attributes error: argument 1 error, " + err.Error())
	}
	// arg2: attributes
	// first get number
	num, err := utils.ReadVarUint(args)
	if err != nil {
		return utils.BYTE_FALSE, errors.New("register ID with attributes error: argument 2 error, " + err.Error())
	}
	// next parse each attribute
	var arg2 = make([]attribute, 0)
	for i := 0; i < int(num); i++ {
		var v attribute
		err = v.Deserialize(args)
		if err != nil {
			return utils.BYTE_FALSE, errors.New("register ID with attributes error: argument 2 error, " + err.Error())
		}
		arg2 = append(arg2, v)
	}

	key, err := encodeID(arg0)
	if err != nil {
		return utils.BYTE_FALSE, errors.New("register ID with attributes error: " + err.Error())
	}

	if checkIDExistence(srvc, key) {
		return utils.BYTE_FALSE, errors.New("register ID with attributes error: already registered")
	}
	public, err := keypair.DeserializePublicKey(arg1)
	if err != nil {
		return utils.BYTE_FALSE, errors.New("register ID with attributes error: invalid public key: " + err.Error())
	}
	addr := types.AddressFromPubKey(public)
	if !srvc.ContextRef.CheckWitness(addr) {
		return utils.BYTE_FALSE, errors.New("register ID with attributes error: check witness failed")
	}

	_, err = insertPk(srvc, key, arg1)
	if err != nil {
		return utils.BYTE_FALSE, errors.New("register ID with attributes error: store pubic key error: " + err.Error())
	}

	err = batchInsertAttr(srvc, key, arg2)
	if err != nil {
		return utils.BYTE_FALSE, errors.New("register ID with attributes error: insert attribute error: " + err.Error())
	}

	srvc.CacheDB.Put(key, states.GenRawStorageItem([]byte{flag_exist}))
	triggerRegisterEvent(srvc, arg0)
	return utils.BYTE_TRUE, nil
}

func addKey(srvc *native.NativeService) ([]byte, error) {
	log.Debug("ID contract: AddKey")
	args := bytes.NewBuffer(srvc.Input)
	// arg0: id
	arg0, err := serialization.ReadVarBytes(args)
	if err != nil {
		return utils.BYTE_FALSE, errors.New("add key failed: argument 0 error, " + err.Error())
	}
	log.Debug("arg 0:", hex.EncodeToString(arg0))

	// arg1: public key
	arg1, err := serialization.ReadVarBytes(args)
	if err != nil {
		return utils.BYTE_FALSE, errors.New("add key failed: argument 1 error, " + err.Error())
	}
	log.Debug("arg 1:", hex.EncodeToString(arg1))

	// arg2: operator's public key
	arg2, err := serialization.ReadVarBytes(args)
	if err != nil {
		return utils.BYTE_FALSE, errors.New("add key failed: argument 2 error, " + err.Error())
	}
	log.Debug("arg 2:", hex.EncodeToString(arg2))

	if err = checkWitness(srvc, arg2); err != nil {
		return utils.BYTE_FALSE, errors.New("add key failed: check witness failed, " + err.Error())
	}

	key, err := encodeID(arg0)
	if err != nil {
		return utils.BYTE_FALSE, errors.New("add key failed: " + err.Error())
	}
	if !isOwner(srvc, key, arg2) {
		return utils.BYTE_FALSE, errors.New("add key failed: operator has no authorization")
	}

	item, _, err := findPk(srvc, key, arg1)
	if item != 0 {
		return utils.BYTE_FALSE, errors.New("add key failed: already exists")
	}

	keyID, err := insertPk(srvc, key, arg1)
	if err != nil {
		return utils.BYTE_FALSE, errors.New("add key failed: insert public key error, " + err.Error())
	}

	triggerPublicEvent(srvc, "add", arg0, arg1, keyID)

	return utils.BYTE_TRUE, nil
}

func removeKey(srvc *native.NativeService) ([]byte, error) {
	args := bytes.NewBuffer(srvc.Input)
	// arg0: id
	arg0, err := serialization.ReadVarBytes(args)
	if err != nil {
		return utils.BYTE_FALSE, fmt.Errorf("remove key failed: argument 0 error, %s", err)
	}

	// arg1: public key
	arg1, err := serialization.ReadVarBytes(args)
	if err != nil {
		return utils.BYTE_FALSE, fmt.Errorf("remove key failed: argument 1 error, %s", err)
	}

	// arg2: operator's public key / address
	arg2, err := serialization.ReadVarBytes(args)
	if err != nil {
		return utils.BYTE_FALSE, fmt.Errorf("remove key failed: argument 2 error, %s", err)
	}
	if err = checkWitness(srvc, arg2); err != nil {
		return utils.BYTE_FALSE, fmt.Errorf("remove key failed: check witness failed, %s", err)
	}

	key, err := encodeID(arg0)
	if err != nil {
		return utils.BYTE_FALSE, fmt.Errorf("remove key failed: %s", err)
	}
	if !checkIDExistence(srvc, key) {
		return utils.BYTE_FALSE, errors.New("remove key failed: ID not registered")
	}
	if !isOwner(srvc, key, arg2) {
		return utils.BYTE_FALSE, errors.New("remove key failed: operator has no authorization")
	}

	keyID, err := revokePk(srvc, key, arg1)
	if err != nil {
		return utils.BYTE_FALSE, fmt.Errorf("remove key failed: %s", err)
	}

	triggerPublicEvent(srvc, "remove", arg0, arg1, keyID)

	return utils.BYTE_TRUE, nil
}

func addAttributes(srvc *native.NativeService) ([]byte, error) {
	args := bytes.NewBuffer(srvc.Input)
	// arg0: ID
	arg0, err := serialization.ReadVarBytes(args)
	if err != nil {
		return utils.BYTE_FALSE, fmt.Errorf("add attributes failed, argument 0 error: %s", err)
	}
	// arg1: attributes
	// first get number
	num, err := utils.ReadVarUint(args)
	if err != nil {
		return utils.BYTE_FALSE, fmt.Errorf("add attributes failed, argument 1 error: %s", err)
	}
	// next parse each attribute
	var arg1 = make([]attribute, 0)
	for i := 0; i < int(num); i++ {
		var v attribute
		err = v.Deserialize(args)
		if err != nil {
			return utils.BYTE_FALSE, fmt.Errorf("add attributes failed, argument 1 error: %s", err)
		}
		arg1 = append(arg1, v)
	}
	// arg2: opperator's public key
	arg2, err := serialization.ReadVarBytes(args)
	if err != nil {
		return utils.BYTE_FALSE, fmt.Errorf("add attributes failed, argument 2 error: %s", err)
	}

	key, err := encodeID(arg0)
	if err != nil {
		return utils.BYTE_FALSE, fmt.Errorf("add attributes failed: %s", err)
	}
	if !checkIDExistence(srvc, key) {
		return utils.BYTE_FALSE, errors.New("add attributes failed, ID not registered")
	}
	if !isOwner(srvc, key, arg2) {
		return utils.BYTE_FALSE, errors.New("add attributes failed, no authorization")
	}
	err = checkWitness(srvc, arg2)
	if err != nil {
		return utils.BYTE_FALSE, fmt.Errorf("add attributes failed, %s", err)
	}

	err = batchInsertAttr(srvc, key, arg1)
	if err != nil {
		return utils.BYTE_FALSE, fmt.Errorf("add attributes failed, %s", err)
	}

	paths := getAttrKeys(arg1)
	triggerAttributeEvent(srvc, "add", arg0, paths)
	return utils.BYTE_TRUE, nil
}

func removeAttribute(srvc *native.NativeService) ([]byte, error) {
	args := bytes.NewBuffer(srvc.Input)
	// arg0: ID
	arg0, err := serialization.ReadVarBytes(args)
	if err != nil {
		return utils.BYTE_FALSE, errors.New("remove attribute failed: argument 0 error")
	}
	// arg1: path
	arg1, err := serialization.ReadVarBytes(args)
	if err != nil {
		return utils.BYTE_FALSE, errors.New("remove attribute failed: argument 1 error")
	}
	// arg2: operator's public key
	arg2, err := serialization.ReadVarBytes(args)
	if err != nil {
		return utils.BYTE_FALSE, errors.New("remove attribute failed: argument 2 error")
	}

	err = checkWitness(srvc, arg2)
	if err != nil {
		return utils.BYTE_FALSE, errors.New("remove attribute failed: " + err.Error())
	}
	key, err := encodeID(arg0)
	if err != nil {
		return utils.BYTE_FALSE, errors.New("remove attribute failed: " + err.Error())
	}
	if !checkIDExistence(srvc, key) {
		return utils.BYTE_FALSE, errors.New("remove attribute failed: ID not registered")
	}
	if !isOwner(srvc, key, arg2) {
		return utils.BYTE_FALSE, errors.New("remove attribute failed: no authorization")
	}

	err = deleteAttr(srvc, key, arg1)
	if err != nil {
		return utils.BYTE_FALSE, errors.New("remove attribute failed: " + err.Error())
	}

	triggerAttributeEvent(srvc, "remove", arg0, [][]byte{arg1})
	return utils.BYTE_TRUE, nil
}

func verifySignature(srvc *native.NativeService) ([]byte, error) {
	args := bytes.NewBuffer(srvc.Input)
	// arg0: ID
	arg0, err := serialization.ReadVarBytes(args)
	if err != nil {
		return utils.BYTE_FALSE, errors.New("verify signature error: argument 0 error, " + err.Error())
	}
	// arg1: index of public key
	arg1, err := utils.ReadVarUint(args)
	if err != nil {
		return utils.BYTE_FALSE, errors.New("verify signature error: argument 1 error, " + err.Error())
	}

	key, err := encodeID(arg0)
	if err != nil {
		return utils.BYTE_FALSE, errors.New("verify signature error: " + err.Error())
	}
	owner, err := getPk(srvc, key, uint32(arg1))
	if err != nil {
		return utils.BYTE_FALSE, errors.New("verify signature error: get key failed, " + err.Error())
	} else if owner == nil {
		return utils.BYTE_FALSE, errors.New("verify signature error: public key not found")
	}

	err = checkWitness(srvc, owner.key)
	if err != nil {
		return utils.BYTE_FALSE, errors.New("verify signature failed: " + err.Error())
	}

	return utils.BYTE_TRUE, nil
}

func revokeID(srvc *native.NativeService) ([]byte, error) {
	args := bytes.NewBuffer(srvc.Input)
	// arg0: id
	arg0, err := serialization.ReadVarBytes(args)
	if err != nil {
		return utils.BYTE_FALSE, fmt.Errorf("argument 0 error")
	}
	// arg1: index of public key
	arg1, err := utils.ReadVarUint(args)
	if err != nil {
		return utils.BYTE_FALSE, fmt.Errorf("argument 1 error")
	}

	encID, err := encodeID(arg0)
	if err != nil {
		return utils.BYTE_FALSE, err
	}

	if !checkIDExistence(srvc, encID) {
		return utils.BYTE_FALSE, fmt.Errorf("%s is not registered or already revoked", string(arg0))
	}

	pk, err := getPk(srvc, encID, uint32(arg1))
	if err != nil {
		return utils.BYTE_FALSE, fmt.Errorf("get public key error: %s", err)
	}

	if checkWitness(srvc, pk.key) != nil {
		return utils.BYTE_FALSE, fmt.Errorf("authorization failed")
	}

	deleteID(srvc, encID)
	newEvent(srvc, []interface{}{"Revoke", string(arg0)})
	return utils.BYTE_TRUE, nil
}
