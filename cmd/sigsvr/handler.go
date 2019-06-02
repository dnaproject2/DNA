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

package sigsvr

import "github.com/dnaproject2/DNA/cmd/sigsvr/handlers"

func init() {
	DefCliRpcSvr.RegHandler("createaccount", handlers.CreateAccount)
	DefCliRpcSvr.RegHandler("exportaccount", handlers.ExportAccount)
	DefCliRpcSvr.RegHandler("sigdata", handlers.SigData)
	DefCliRpcSvr.RegHandler("sigrawtx", handlers.SigRawTransaction)
	DefCliRpcSvr.RegHandler("sigmutilrawtx", handlers.SigMutilRawTransaction)
	DefCliRpcSvr.RegHandler("sigtransfertx", handlers.SigTransferTransaction)
	DefCliRpcSvr.RegHandler("signeovminvoketx", handlers.SigNeoVMInvokeTx)
	DefCliRpcSvr.RegHandler("signeovminvokeabitx", handlers.SigNeoVMInvokeAbiTx)
	DefCliRpcSvr.RegHandler("signativeinvoketx", handlers.SigNativeInvokeTx)
}
