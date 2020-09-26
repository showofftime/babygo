package main

import (
	"syscall"
)

var debugCodeGen bool

func emitComment(indent int, format string, a ...string) {
	if !debugCodeGen {
		return
	}
	var spaces []uint8
	var i int
	for i = 0; i < indent; i++ {
		spaces = append(spaces, ' ')
	}
	var format2 = string(spaces) + "# " + format
	var s = fmtSprintf(format2, a)
	syscall.Write(1, []uint8(s))
}

func evalInt(expr *astExpr) int {
	switch expr.dtype {
	case "*astBasicLit":
		return Atoi(expr.basicLit.Value)
	}
	return 0
}

func emitPopBool(comment string) {
	fmtPrintf("  popq %%rax # result of %s\n", comment)
}

func emitPopAddress(comment string) {
	fmtPrintf("  popq %%rax # address of %s\n", comment)
}

func emitPopString() {
	fmtPrintf("  popq %%rax # string.ptr\n")
	fmtPrintf("  popq %%rcx # string.len\n")
}

func emitPopSlice() {
	fmtPrintf("  popq %%rax # slice.ptr\n")
	fmtPrintf("  popq %%rcx # slice.len\n")
	fmtPrintf("  popq %%rdx # slice.cap\n")
}

func emitPushStackTop(condType *Type, comment string) {
	switch kind(condType) {
	case T_STRING:
		fmtPrintf("  movq 8(%%rsp), %%rcx # copy str.len from stack top (%s)\n", comment)
		fmtPrintf("  movq 0(%%rsp), %%rax # copy str.ptr from stack top (%s)\n", comment)
		fmtPrintf("  pushq %%rcx # str.len\n")
		fmtPrintf("  pushq %%rax # str.ptr\n")
	case T_POINTER, T_UINTPTR, T_BOOL, T_INT, T_UINT8, T_UINT16:
		fmtPrintf("  movq (%%rsp), %%rax # copy stack top value (%s) \n", comment)
		fmtPrintf("  pushq %%rax\n")
	default:
		throw(kind(condType))
	}
}

func emitRevertStackPointer(size int) {
	fmtPrintf("  addq $%s, %%rsp # revert stack pointer\n", Itoa(size))
}

func emitAddConst(addValue int, comment string) {
	emitComment(2, "Add const: %s\n", comment)
	fmtPrintf("  popq %%rax\n")
	fmtPrintf("  addq $%s, %%rax\n", Itoa(addValue))
	fmtPrintf("  pushq %%rax\n")
}

func emitLoad(t *Type) {
	if t == nil {
		panic2(__func__, "nil type error\n")
	}
	emitPopAddress(kind(t))
	switch kind(t) {
	case T_SLICE:
		fmtPrintf("  movq %d(%%rax), %%rdx\n", Itoa(16))
		fmtPrintf("  movq %d(%%rax), %%rcx\n", Itoa(8))
		fmtPrintf("  movq %d(%%rax), %%rax\n", Itoa(0))
		fmtPrintf("  pushq %%rdx # cap\n")
		fmtPrintf("  pushq %%rcx # len\n")
		fmtPrintf("  pushq %%rax # ptr\n")
	case T_STRING:
		fmtPrintf("  movq %d(%%rax), %%rdx\n", Itoa(8))
		fmtPrintf("  movq %d(%%rax), %%rax\n", Itoa(0))
		fmtPrintf("  pushq %%rdx # len\n")
		fmtPrintf("  pushq %%rax # ptr\n")
	case T_UINT8:
		fmtPrintf("  movzbq %d(%%rax), %%rax # load uint8\n", Itoa(0))
		fmtPrintf("  pushq %%rax\n")
	case T_UINT16:
		fmtPrintf("  movzwq %d(%%rax), %%rax # load uint16\n", Itoa(0))
		fmtPrintf("  pushq %%rax\n")
	case T_INT, T_BOOL, T_UINTPTR, T_POINTER:
		fmtPrintf("  movq %d(%%rax), %%rax # load int\n", Itoa(0))
		fmtPrintf("  pushq %%rax\n")
	case T_ARRAY:
		// pure proxy
		fmtPrintf("  pushq %%rax\n")
	default:
		panic2(__func__, "TBI:kind="+kind(t))
	}
}

func emitVariableAddr(variable *Variable) {
	emitComment(2, "emit Addr of variable \"%s\" \n", variable.name)

	if variable.isGlobal {
		fmtPrintf("  leaq %s(%%rip), %%rax # global variable addr\n", variable.globalSymbol)
	} else {
		fmtPrintf("  leaq %d(%%rbp), %%rax # local variable addr\n", Itoa(variable.localOffset))
	}

	fmtPrintf("  pushq %%rax\n")
}

func emitListHeadAddr(list *astExpr) {
	var t = getTypeOfExpr(list)
	switch kind(t) {
	case T_ARRAY:
		emitAddr(list) // array head
	case T_SLICE:
		emitExpr(list, nil)
		emitPopSlice()
		fmtPrintf("  pushq %%rax # slice.ptr\n")
	case T_STRING:
		emitExpr(list, nil)
		emitPopString()
		fmtPrintf("  pushq %%rax # string.ptr\n")
	default:
		panic2(__func__, "kind="+kind(getTypeOfExpr(list)))
	}
}

func emitAddr(expr *astExpr) {
	emitComment(2, "[emitAddr] %s\n", expr.dtype)
	switch expr.dtype {
	case "*astIdent":
		if expr.ident.Obj.Kind == astVar {
			assert(expr.ident.Obj.Variable != nil,
				"ERROR: Variable is nil for name : "+expr.ident.Obj.Name, __func__)
			emitVariableAddr(expr.ident.Obj.Variable)
		} else {
			panic2(__func__, "Unexpected Kind "+expr.ident.Obj.Kind)
		}
	case "*astIndexExpr":
		emitExpr(expr.indexExpr.Index, nil) // index number
		var list = expr.indexExpr.X
		var elmType = getTypeOfExpr(expr)
		emitListElementAddr(list, elmType)
	case "*astStarExpr":
		emitExpr(expr.starExpr.X, nil)
	case "*astSelectorExpr": // X.Sel
		var typeOfX = getTypeOfExpr(expr.selectorExpr.X)
		var structType *Type
		switch kind(typeOfX) {
		case T_STRUCT:
			// strct.field
			structType = typeOfX
			emitAddr(expr.selectorExpr.X)
		case T_POINTER:
			// ptr.field
			assert(typeOfX.e.dtype == "*astStarExpr", "should be *astStarExpr", __func__)
			var ptrType = typeOfX.e.starExpr
			structType = e2t(ptrType.X)
			emitExpr(expr.selectorExpr.X, nil)
		default:
			panic2(__func__, "TBI:"+kind(typeOfX))
		}
		var field = lookupStructField(getStructTypeSpec(structType), expr.selectorExpr.Sel.Name)
		var offset = getStructFieldOffset(field)
		emitAddConst(offset, "struct head address + struct.field offset")
	default:
		panic2(__func__, "TBI "+expr.dtype)
	}
}

func isType(expr *astExpr) bool {
	switch expr.dtype {
	case "*astArrayType":
		return true
	case "*astIdent":
		if expr.ident == nil {
			panic2(__func__, "ident should not be nil")
		}
		if expr.ident.Obj == nil {
			panic2(__func__, " unresolved ident:"+expr.ident.Name)
		}
		emitComment(0, "[isType][DEBUG] expr.ident.Name = %s\n", expr.ident.Name)
		emitComment(0, "[isType][DEBUG] expr.ident.Obj = %s,%s\n",
			expr.ident.Obj.Name, expr.ident.Obj.Kind)
		return expr.ident.Obj.Kind == astType
	case "*astParenExpr":
		return isType(expr.parenExpr.X)
	case "*astStarExpr":
		return isType(expr.starExpr.X)
	default:
		emitComment(0, "[isType][%s] is not considered a type\n", expr.dtype)
	}

	return false

}

func emitConversion(tp *Type, arg0 *astExpr) {
	emitComment(0, "[emitConversion]\n")
	var typeExpr = tp.e
	switch typeExpr.dtype {
	case "*astIdent":
		switch typeExpr.ident.Obj {
		case gString: // string(e)
			switch kind(getTypeOfExpr(arg0)) {
			case T_SLICE: // string(slice)
				emitExpr(arg0, nil) // slice
				emitPopSlice()
				fmtPrintf("  pushq %%rcx # str len\n")
				fmtPrintf("  pushq %%rax # str ptr\n")
			}
		case gInt, gUint8, gUint16, gUintptr: // int(e)
			emitComment(0, "[emitConversion] to int \n")
			emitExpr(arg0, nil)
		default:
			panic2(__func__, "[*astIdent] TBI : "+typeExpr.ident.Obj.Name)
		}
	case "*astArrayType": // Conversion to slice
		var arrayType = typeExpr.arrayType
		if arrayType.Len != nil {
			panic2(__func__, "internal error")
		}
		if (kind(getTypeOfExpr(arg0))) != T_STRING {
			panic2(__func__, "source type should be string")
		}
		emitComment(2, "Conversion of string => slice \n")
		emitExpr(arg0, nil)
		emitPopString()
		fmtPrintf("  pushq %%rcx # cap\n")
		fmtPrintf("  pushq %%rcx # len\n")
		fmtPrintf("  pushq %%rax # ptr\n")
	case "*astParenExpr":
		emitConversion(e2t(typeExpr.parenExpr.X), arg0)
	case "*astStarExpr": // (*T)(e)
		emitComment(0, "[emitConversion] to pointer \n")
		emitExpr(arg0, nil)
	default:
		panic2(__func__, "TBI :"+typeExpr.dtype)
	}
}

func emitZeroValue(t *Type) {
	switch kind(t) {
	case T_SLICE:
		fmtPrintf("  pushq $0 # slice zero value\n")
		fmtPrintf("  pushq $0 # slice zero value\n")
		fmtPrintf("  pushq $0 # slice zero valuer\n")
	case T_STRING:
		fmtPrintf("  pushq $0 # string zero value\n")
		fmtPrintf("  pushq $0 # string zero value\n")
	case T_INT, T_UINTPTR, T_UINT8, T_POINTER, T_BOOL:
		fmtPrintf("  pushq $0 # %s zero value\n", kind(t))
	case T_STRUCT:
		//@FIXME
	default:
		panic2(__func__, "TBI:"+kind(t))
	}
}

func emitLen(arg *astExpr) {
	emitComment(0, "[%s] begin\n", __func__)
	switch kind(getTypeOfExpr(arg)) {
	case T_ARRAY:
		var typ = getTypeOfExpr(arg)
		var arrayType = typ.e.arrayType
		emitExpr(arrayType.Len, nil)
	case T_SLICE:
		emitExpr(arg, nil)
		emitPopSlice()
		fmtPrintf("  pushq %%rcx # len\n")
	case T_STRING:
		emitExpr(arg, nil)
		emitPopString()
		fmtPrintf("  pushq %%rcx # len\n")
	default:
		throw(kind(getTypeOfExpr(arg)))
	}
	emitComment(0, "[%s] end\n", __func__)
}

func emitCap(arg *astExpr) {
	switch kind(getTypeOfExpr(arg)) {
	case T_ARRAY:
		var typ = getTypeOfExpr(arg)
		var arrayType = typ.e.arrayType
		emitExpr(arrayType.Len, nil)
	case T_SLICE:
		emitExpr(arg, nil)
		emitPopSlice()
		fmtPrintf("  pushq %%rdx # cap\n")
	case T_STRING:
		panic("cap() cannot accept string type")
	default:
		throw(kind(getTypeOfExpr(arg)))
	}
}

func emitCallMalloc(size int) {
	fmtPrintf("  pushq $%s\n", Itoa(size))
	// call malloc and return pointer
	fmtPrintf("  callq runtime.malloc\n") // no need to invert args orders
	emitRevertStackPointer(intSize)
	fmtPrintf("  pushq %%rax # addr\n")
}

func emitArrayLiteral(arrayType *astArrayType, arrayLen int, elts []*astExpr) {
	var elmType = e2t(arrayType.Elt)
	var elmSize = getSizeOfType(elmType)
	var memSize = elmSize * arrayLen
	emitCallMalloc(memSize) // push
	var i int
	var elm *astExpr
	for i, elm = range elts {
		// emit lhs
		emitPushStackTop(tUintptr, "malloced address")
		emitAddConst(elmSize*i, "malloced address + elmSize * index ("+Itoa(i)+")")
		emitExpr(elm, elmType)
		emitStore(elmType)
	}
}

func emitInvertBoolValue() {
	emitPopBool("")
	fmtPrintf("  xor $1, %%rax\n")
	fmtPrintf("  pushq %%rax\n")
}

func emitTrue() {
	fmtPrintf("  pushq $1 # true\n")
}

func emitFalse() {
	fmtPrintf("  pushq $0 # false\n")
}

type Arg struct {
	e      *astExpr
	t      *Type // expected type
	offset int
}

func emitArgs(args []*Arg) int {
	var totalPushedSize int
	//var arg *astExpr
	var arg *Arg
	for _, arg = range args {
		var t *Type
		if arg.t != nil {
			t = arg.t
		} else {
			t = getTypeOfExpr(arg.e)
		}
		arg.offset = totalPushedSize
		totalPushedSize = totalPushedSize + getPushSizeOfType(t)
	}
	fmtPrintf("  subq $%d, %%rsp # for args\n", Itoa(totalPushedSize))
	for _, arg = range args {
		emitExpr(arg.e, arg.t)
	}
	fmtPrintf("  addq $%d, %%rsp # for args\n", Itoa(totalPushedSize))

	for _, arg = range args {
		var t *Type
		if arg.t != nil {
			t = arg.t
		} else {
			t = getTypeOfExpr(arg.e)
		}
		switch kind(t) {
		case T_BOOL, T_INT, T_UINT8, T_POINTER, T_UINTPTR:
			fmtPrintf("  movq %d-8(%%rsp) , %%rax # load\n", Itoa(-arg.offset))
			fmtPrintf("  movq %%rax, %d(%%rsp) # store\n", Itoa(+arg.offset))
		case T_STRING:
			fmtPrintf("  movq %d-16(%%rsp), %%rax\n", Itoa(-arg.offset))
			fmtPrintf("  movq %d-8(%%rsp), %%rcx\n", Itoa(-arg.offset))
			fmtPrintf("  movq %%rax, %d(%%rsp)\n", Itoa(+arg.offset))
			fmtPrintf("  movq %%rcx, %d+8(%%rsp)\n", Itoa(+arg.offset))
		case T_SLICE:
			fmtPrintf("  movq %d-24(%%rsp), %%rax\n", Itoa(-arg.offset)) // arg1: slc.ptr
			fmtPrintf("  movq %d-16(%%rsp), %%rcx\n", Itoa(-arg.offset)) // arg1: slc.len
			fmtPrintf("  movq %d-8(%%rsp), %%rdx\n", Itoa(-arg.offset))  // arg1: slc.cap
			fmtPrintf("  movq %%rax, %d+0(%%rsp)\n", Itoa(+arg.offset))  // arg1: slc.ptr
			fmtPrintf("  movq %%rcx, %d+8(%%rsp)\n", Itoa(+arg.offset))  // arg1: slc.len
			fmtPrintf("  movq %%rdx, %d+16(%%rsp)\n", Itoa(+arg.offset)) // arg1: slc.cap
		default:
			throw(kind(t))
		}
	}

	return totalPushedSize
}

func emitCallNonDecl(symbol string, eArgs []*astExpr) {
	var args []*Arg
	var eArg *astExpr
	for _, eArg = range eArgs {
		var arg = new(Arg)
		arg.e = eArg
		arg.t = nil
		args = append(args, arg)
	}
	emitCall(symbol, args)
}

func emitCall(symbol string, args []*Arg) {
	emitComment(0, "[%s] %s\n", __func__, symbol)
	var totalPushedSize = emitArgs(args)
	fmtPrintf("  callq %s\n", symbol)
	emitRevertStackPointer(totalPushedSize)
}

func emitFuncall(fun *astExpr, eArgs []*astExpr) {
	switch fun.dtype {
	case "*astIdent":
		emitComment(0, "[%s][*astIdent]\n", __func__)
		var fnIdent = fun.ident
		switch fnIdent.Obj {
		case gLen:
			var arg = eArgs[0]
			emitLen(arg)
			return
		case gCap:
			var arg = eArgs[0]
			emitCap(arg)
			return
		case gNew:
			var typeArg = e2t(eArgs[0])
			var size = getSizeOfType(typeArg)
			emitCallMalloc(size)
			return
		case gMake:
			var typeArg = e2t(eArgs[0])
			switch kind(typeArg) {
			case T_SLICE:
				// make([]T, ...)
				var arrayType = typeArg.e.arrayType
				//assert(ok, "should be *ast.ArrayType")
				var elmSize = getSizeOfType(e2t(arrayType.Elt))
				var numlit = newNumberLiteral(elmSize)
				var eNumLit = new(astExpr)
				eNumLit.dtype = "*astBasicLit"
				eNumLit.basicLit = numlit
				var args []*Arg
				var arg0 *Arg // elmSize
				var arg1 *Arg
				var arg2 *Arg
				arg0 = new(Arg)
				arg0.e = eNumLit
				arg0.t = tInt
				arg1 = new(Arg) // len
				arg1.e = eArgs[1]
				arg1.t = tInt
				arg2 = new(Arg) // cap
				arg2.e = eArgs[2]
				arg2.t = tInt
				args = append(args, arg0)
				args = append(args, arg1)
				args = append(args, arg2)

				emitCall("runtime.makeSlice", args)
				fmtPrintf("  pushq %%rsi # slice cap\n")
				fmtPrintf("  pushq %%rdi # slice len\n")
				fmtPrintf("  pushq %%rax # slice ptr\n")
				return
			default:
				panic2(__func__, "TBI")
			}

			return
		case gAppend:
			var sliceArg = eArgs[0]
			var elemArg = eArgs[1]
			var elmType = getElementTypeOfListType(getTypeOfExpr(sliceArg))
			var elmSize = getSizeOfType(elmType)

			var args []*Arg
			var arg0 = new(Arg) // slice
			arg0.e = sliceArg
			args = append(args, arg0)

			var arg1 = new(Arg) // element
			arg1.e = elemArg
			arg1.t = elmType
			args = append(args, arg1)

			var symbol string
			switch elmSize {
			case 1:
				symbol = "runtime.append1"
			case 8:
				symbol = "runtime.append8"
			case 16:
				symbol = "runtime.append16"
			case 24:
				symbol = "runtime.append24"
			default:
				panic2(__func__, "Unexpected elmSize")
			}
			emitCall(symbol, args)
			fmtPrintf("  pushq %%rsi # slice cap\n")
			fmtPrintf("  pushq %%rdi # slice len\n")
			fmtPrintf("  pushq %%rax # slice ptr\n")
			return
		}

		var fn = fun.ident
		if fn.Name == "print" {
			emitExpr(eArgs[0], nil)
			fmtPrintf("  callq runtime.printstring\n")
			fmtPrintf("  addq $%s, %%rsp # revert \n", Itoa(16))
			return
		}

		if fn.Name == "makeSlice1" || fn.Name == "makeSlice8" || fn.Name == "makeSlice16" || fn.Name == "makeSlice24" {
			fn.Name = "makeSlice"
		}
		// general function call
		var symbol = pkg.name + "." + fn.Name
		emitComment(0, "[%s][*astIdent][default] start\n", __func__)

		var obj = fn.Obj
		var decl = obj.Decl
		if decl == nil {
			panic2(__func__, "[*astCallExpr] decl is nil")
		}
		if decl.dtype != "*astFuncDecl" {
			panic2(__func__, "[*astCallExpr] decl.dtype is invalid")
		}
		var fndecl = decl.funcDecl
		if fndecl == nil {
			panic2(__func__, "[*astCallExpr] fndecl is nil")
		}
		if fndecl.Type == nil {
			panic2(__func__, "[*astCallExpr] fndecl.Type is nil")
		}

		var params = fndecl.Type.params.List
		var variadicArgs []*astExpr
		var variadicElp *astEllipsis
		var args []*Arg
		var eArg *astExpr
		var param *astField
		var argIndex int
		var arg *Arg
		var lenParams = len(params)
		for argIndex, eArg = range eArgs {
			emitComment(0, "[%s][*astIdent][default] loop idx %s, len params %s\n", __func__, Itoa(argIndex), Itoa(lenParams))
			if argIndex < lenParams {
				param = params[argIndex]
				if param.Type.dtype == "*astEllipsis" {
					variadicElp = param.Type.ellipsis
					variadicArgs = make([]*astExpr, 0, 20)
				}
			}
			if variadicElp != nil {
				variadicArgs = append(variadicArgs, eArg)
				continue
			}

			var paramType = e2t(param.Type)
			arg = new(Arg)
			arg.e = eArg
			arg.t = paramType
			args = append(args, arg)
		}

		if variadicElp != nil {
			// collect args as a slice
			var sliceType = new(astArrayType)
			sliceType.Elt = variadicElp.Elt
			var eSliceType = new(astExpr)
			eSliceType.dtype = "*astArrayType"
			eSliceType.arrayType = sliceType
			var sliceLiteral = new(astCompositeLit)
			sliceLiteral.Type = eSliceType
			sliceLiteral.Elts = variadicArgs
			var eSliceLiteral = new(astExpr)
			eSliceLiteral.compositeLit = sliceLiteral
			eSliceLiteral.dtype = "*astCompositeLit"
			var _arg = new(Arg)
			_arg.e = eSliceLiteral
			_arg.t = e2t(eSliceType)
			args = append(args, _arg)
		} else if len(args) < len(params) {
			// Add nil as a variadic arg
			emitComment(0, "len(args)=%s, len(params)=%s\n", Itoa(len(args)), Itoa(len(params)))
			var param = params[len(args)]
			if param == nil {
				panic2(__func__, "param should not be nil")
			}
			if param.Type == nil {
				panic2(__func__, "param.Type should not be nil")
			}
			assert(param.Type.dtype == "*astEllipsis", "internal error", __func__)

			var _arg = new(Arg)
			_arg.e = exprNil
			_arg.t = e2t(param.Type)
			args = append(args, _arg)
		}

		emitCall(symbol, args)

		// push results
		var results = fndecl.Type.results
		if fndecl.Type.results == nil {
			emitComment(0, "[emitExpr] %s sig.results is nil\n", fn.Name)
		} else {
			emitComment(0, "[emitExpr] %s sig.results.List = %s\n", fn.Name, Itoa(len(fndecl.Type.results.List)))
		}

		if results != nil && len(results.List) == 1 {
			var retval0 = fndecl.Type.results.List[0]
			var knd = kind(e2t(retval0.Type))
			switch knd {
			case T_STRING:
				emitComment(2, "fn.Obj=%s\n", obj.Name)
				fmtPrintf("  pushq %%rdi # str len\n")
				fmtPrintf("  pushq %%rax # str ptr\n")
			case T_BOOL, T_INT, T_UINTPTR, T_POINTER:
				emitComment(2, "fn.Obj=%s\n", obj.Name)
				fmtPrintf("  pushq %%rax\n")
			case T_SLICE:
				fmtPrintf("  pushq %%rsi # slice cap\n")
				fmtPrintf("  pushq %%rdi # slice len\n")
				fmtPrintf("  pushq %%rax # slice ptr\n")
			default:
				panic2(__func__, "Unexpected kind="+knd)
			}
		} else {
			emitComment(2, "No results\n")
		}
		return
	case "*astSelectorExpr":
		var selectorExpr = fun.selectorExpr
		if selectorExpr.X.dtype != "*astIdent" {
			panic2(__func__, "TBI selectorExpr.X.dtype="+selectorExpr.X.dtype)
		}
		var symbol string = selectorExpr.X.ident.Name + "." + selectorExpr.Sel.Name
		switch symbol {
		case "os.Exit":
			emitCallNonDecl(symbol, eArgs)
		case "syscall.Write":
			emitCallNonDecl(symbol, eArgs)
		case "syscall.Open":
			// func decl is in runtime
			emitCallNonDecl(symbol, eArgs)
			fmtPrintf("  pushq %%rax # fd\n")
		case "syscall.Read":
			// func decl is in runtime
			emitCallNonDecl(symbol, eArgs)
			fmtPrintf("  pushq %%rax # fd\n")
		case "syscall.Syscall":
			emitCallNonDecl(symbol, eArgs)
			fmtPrintf("  pushq %%rax # ret\n")
		case "unsafe.Pointer":
			emitExpr(eArgs[0], nil)
		default:
			fmtPrintf("  callq %s.%s\n", selectorExpr.X.ident.Name, selectorExpr.Sel.Name)
			panic2(__func__, "[*astSelectorExpr] Unsupported call to "+symbol)
		}
	case "*astParenExpr":
		panic2(__func__, "[astParenExpr] TBI ")
	default:
		panic2(__func__, "TBI fun.dtype="+fun.dtype)
	}
}

func emitExpr(e *astExpr, forceType *Type) {
	emitComment(2, "[emitExpr] dtype=%s\n", e.dtype)
	switch e.dtype {
	case "*astIdent":
		var ident = e.ident
		if ident.Obj == nil {
			panic2(__func__, "ident unresolved:"+ident.Name)
		}
		switch e.ident.Obj {
		case gTrue:
			emitTrue()
			return
		case gFalse:
			emitFalse()
			return
		case gNil:
			if forceType == nil {
				panic2(__func__, "Type is required to emit nil")
			}
			switch kind(forceType) {
			case T_SLICE, T_POINTER:
				emitZeroValue(forceType)
			default:
				panic2(__func__, "Unexpected kind="+kind(forceType))
			}
			return
		}
		switch ident.Obj.Kind {
		case astVar:
			emitAddr(e)
			var t = getTypeOfExpr(e)
			emitLoad(t)
		case astConst:
			var valSpec = ident.Obj.Decl.valueSpec
			assert(valSpec != nil, "valSpec should not be nil", __func__)
			assert(valSpec.Value != nil, "valSpec should not be nil", __func__)
			assert(valSpec.Value.dtype == "*astBasicLit", "const value should be a literal", __func__)
			var t *Type
			if valSpec.Type != nil {
				t = e2t(valSpec.Type)
			} else {
				t = forceType
			}
			emitExpr(valSpec.Value, t)
		case astType:
			panic2(__func__, "[*astIdent] Kind Typ should not come here")
		default:
			panic2(__func__, "[*astIdent] unknown Kind="+ident.Obj.Kind+" Name="+ident.Obj.Name)
		}
	case "*astIndexExpr":
		emitAddr(e)
		emitLoad(getTypeOfExpr(e))
	case "*astStarExpr":
		emitAddr(e)
		emitLoad(getTypeOfExpr(e))
	case "*astSelectorExpr":
		emitAddr(e)
		emitLoad(getTypeOfExpr(e))
	case "*astBasicLit":
		//		emitComment(0, "basicLit.Kind = %s \n", e.basicLit.Kind)
		switch e.basicLit.Kind {
		case "INT":
			var ival = Atoi(e.basicLit.Value)
			fmtPrintf("  pushq $%d # number literal\n", Itoa(ival))
		case "STRING":
			var sl = getStringLiteral(e.basicLit)
			if sl.strlen == 0 {
				// zero value
				emitZeroValue(tString)
			} else {
				fmtPrintf("  pushq $%d # str len\n", Itoa(sl.strlen))
				fmtPrintf("  leaq %s, %%rax # str ptr\n", sl.label)
				fmtPrintf("  pushq %%rax # str ptr\n")
			}
		case "CHAR":
			var val = e.basicLit.Value
			var char = val[1]
			if val[1] == '\\' {
				switch val[2] {
				case '\'':
					char = '\''
				case 'n':
					char = '\n'
				case '\\':
					char = '\\'
				case 't':
					char = '\t'
				case 'r':
					char = '\r'
				}
			}
			fmtPrintf("  pushq $%d # convert char literal to int\n", Itoa(int(char)))
		default:
			panic2(__func__, "[*astBasicLit] TBI : "+e.basicLit.Kind)
		}
	case "*astCallExpr":
		var fun = e.callExpr.Fun
		emitComment(0, "[%s][*astCallExpr]\n", __func__)
		if isType(fun) {
			emitConversion(e2t(fun), e.callExpr.Args[0])
			return
		}
		emitFuncall(fun, e.callExpr.Args)
	case "*astParenExpr":
		emitExpr(e.parenExpr.X, nil)
	case "*astSliceExpr":
		var list = e.sliceExpr.X
		var listType = getTypeOfExpr(list)
		emitExpr(e.sliceExpr.High, nil)
		emitExpr(e.sliceExpr.Low, nil)
		fmtPrintf("  popq %%rcx # low\n")
		fmtPrintf("  popq %%rax # high\n")
		fmtPrintf("  subq %%rcx, %%rax # high - low\n")
		switch kind(listType) {
		case T_SLICE, T_ARRAY:
			fmtPrintf("  pushq %%rax # cap\n")
			fmtPrintf("  pushq %%rax # len\n")
		case T_STRING:
			fmtPrintf("  pushq %%rax # len\n")
			// no cap
		default:
			panic2(__func__, "Unknown kind="+kind(listType))
		}

		emitExpr(e.sliceExpr.Low, nil)
		var elmType = getElementTypeOfListType(listType)
		emitListElementAddr(list, elmType)
	case "*astUnaryExpr":
		emitComment(0, "[DEBUG] unary op = %s\n", e.unaryExpr.Op)
		switch e.unaryExpr.Op {
		case "+":
			emitExpr(e.unaryExpr.X, nil)
		case "-":
			emitExpr(e.unaryExpr.X, nil)
			fmtPrintf("  popq %%rax # e.X\n")
			fmtPrintf("  imulq $-1, %%rax\n")
			fmtPrintf("  pushq %%rax\n")
		case "&":
			emitAddr(e.unaryExpr.X)
		case "!":
			emitExpr(e.unaryExpr.X, nil)
			emitInvertBoolValue()
		default:
			panic2(__func__, "TBI:astUnaryExpr:"+e.unaryExpr.Op)
		}
	case "*astBinaryExpr":
		if kind(getTypeOfExpr(e.binaryExpr.X)) == T_STRING {
			var args []*Arg
			var argX = new(Arg)
			var argY = new(Arg)
			argX.e = e.binaryExpr.X
			argY.e = e.binaryExpr.Y
			args = append(args, argX)
			args = append(args, argY)
			switch e.binaryExpr.Op {
			case "+":
				emitCall("runtime.catstrings", args)
				fmtPrintf("  pushq %%rdi # slice len\n")
				fmtPrintf("  pushq %%rax # slice ptr\n")
			case "==":
				emitArgs(args)
				emitCompEq(getTypeOfExpr(e.binaryExpr.X))
			case "!=":
				emitArgs(args)
				emitCompEq(getTypeOfExpr(e.binaryExpr.X))
				emitInvertBoolValue()
			default:
				panic2(__func__, "[emitExpr][*astBinaryExpr] string : TBI T_STRING")
			}
			return
		}

		switch e.binaryExpr.Op {
		case "&&":
			labelid++
			var labelExitWithFalse = fmtSprintf(".L.%s.false", []string{Itoa(labelid)})
			var labelExit = fmtSprintf(".L.%d.exit", []string{Itoa(labelid)})
			emitExpr(e.binaryExpr.X, nil) // left
			emitPopBool("left")
			fmtPrintf("  cmpq $1, %%rax\n")
			// exit with false if left is false
			fmtPrintf("  jne %s\n", labelExitWithFalse)

			// if left is true, then eval right and exit
			emitExpr(e.binaryExpr.Y, nil) // right
			fmtPrintf("  jmp %s\n", labelExit)

			fmtPrintf("  %s:\n", labelExitWithFalse)
			emitFalse()
			fmtPrintf("  %s:\n", labelExit)
			return
		case "||":
			labelid++
			var labelExitWithTrue = fmtSprintf(".L.%d.true", []string{Itoa(labelid)})
			var labelExit = fmtSprintf(".L.%d.exit", []string{Itoa(labelid)})
			emitExpr(e.binaryExpr.X, nil) // left
			emitPopBool("left")
			fmtPrintf("  cmpq $1, %%rax\n")
			// exit with true if left is true
			fmtPrintf("  je %s\n", labelExitWithTrue)

			// if left is false, then eval right and exit
			emitExpr(e.binaryExpr.Y, nil) // right
			fmtPrintf("  jmp %s\n", labelExit)

			fmtPrintf("  %s:\n", labelExitWithTrue)
			emitTrue()
			fmtPrintf("  %s:\n", labelExit)
			return
		}

		var t = getTypeOfExpr(e.binaryExpr.X)
		emitExpr(e.binaryExpr.X, nil) // left
		emitExpr(e.binaryExpr.Y, t)   // right
		switch e.binaryExpr.Op {
		case "+":
			fmtPrintf("  popq %%rcx # right\n")
			fmtPrintf("  popq %%rax # left\n")
			fmtPrintf("  addq %%rcx, %%rax\n")
			fmtPrintf("  pushq %%rax\n")
		case "-":
			fmtPrintf("  popq %%rcx # right\n")
			fmtPrintf("  popq %%rax # left\n")
			fmtPrintf("  subq %%rcx, %%rax\n")
			fmtPrintf("  pushq %%rax\n")
		case "*":
			fmtPrintf("  popq %%rcx # right\n")
			fmtPrintf("  popq %%rax # left\n")
			fmtPrintf("  imulq %%rcx, %%rax\n")
			fmtPrintf("  pushq %%rax\n")
		case "%":
			fmtPrintf("  popq %%rcx # right\n")
			fmtPrintf("  popq %%rax # left\n")
			fmtPrintf("  movq $0, %%rdx # init %%rdx\n")
			fmtPrintf("  divq %%rcx\n")
			fmtPrintf("  movq %%rdx, %%rax\n")
			fmtPrintf("  pushq %%rax\n")
		case "/":
			fmtPrintf("  popq %%rcx # right\n")
			fmtPrintf("  popq %%rax # left\n")
			fmtPrintf("  movq $0, %%rdx # init %%rdx\n")
			fmtPrintf("  divq %%rcx\n")
			fmtPrintf("  pushq %%rax\n")
		case "==":
			emitCompEq(t)
		case "!=":
			emitCompEq(t)
			emitInvertBoolValue()
		case "<":
			emitCompExpr("setl")
		case "<=":
			emitCompExpr("setle")
		case ">":
			emitCompExpr("setg")
		case ">=":
			emitCompExpr("setge")
		default:
			panic2(__func__, "# TBI: binary operation for "+e.binaryExpr.Op)
		}
	case "*astCompositeLit":
		// slice , array, map or struct
		var k = kind(e2t(e.compositeLit.Type))
		switch k {
		case T_ARRAY:
			assert(e.compositeLit.Type.dtype == "*astArrayType", "expect *ast.ArrayType", __func__)
			var arrayType = e.compositeLit.Type.arrayType
			var arrayLen = evalInt(arrayType.Len)
			emitArrayLiteral(arrayType, arrayLen, e.compositeLit.Elts)
		case T_SLICE:
			assert(e.compositeLit.Type.dtype == "*astArrayType", "expect *ast.ArrayType", __func__)
			var arrayType = e.compositeLit.Type.arrayType
			var length = len(e.compositeLit.Elts)
			emitArrayLiteral(arrayType, length, e.compositeLit.Elts)
			emitPopAddress("malloc")
			fmtPrintf("  pushq $%d # slice.cap\n", Itoa(length))
			fmtPrintf("  pushq $%d # slice.len\n", Itoa(length))
			fmtPrintf("  pushq %%rax # slice.ptr\n")
		default:
			panic2(__func__, "Unexpected kind="+k)
		}
	default:
		panic2(__func__, "[emitExpr] `TBI:"+e.dtype)
	}
}

func newNumberLiteral(x int) *astBasicLit {
	var r = new(astBasicLit)
	r.Kind = "INT"
	r.Value = Itoa(x)
	return r
}

func emitListElementAddr(list *astExpr, elmType *Type) {
	emitListHeadAddr(list)
	emitPopAddress("list head")
	fmtPrintf("  popq %%rcx # index id\n")
	fmtPrintf("  movq $%s, %%rdx # elm size\n", Itoa(getSizeOfType(elmType)))
	fmtPrintf("  imulq %%rdx, %%rcx\n")
	fmtPrintf("  addq %%rcx, %%rax\n")
	fmtPrintf("  pushq %%rax # addr of element\n")
}

func emitCompEq(t *Type) {
	switch kind(t) {
	case T_STRING:
		fmtPrintf("  callq runtime.cmpstrings\n")
		emitRevertStackPointer(stringSize * 2)
		fmtPrintf("  pushq %%rax # cmp result (1 or 0)\n")
	case T_INT, T_UINT8, T_UINT16, T_UINTPTR, T_POINTER:
		emitCompExpr("sete")
	case T_SLICE:
		emitCompExpr("sete") // @FIXME this is not correct
	default:
		panic2(__func__, "Unexpected kind="+kind(t))
	}
}

//@TODO handle larger types than int
func emitCompExpr(inst string) {
	fmtPrintf("  popq %%rcx # right\n")
	fmtPrintf("  popq %%rax # left\n")
	fmtPrintf("  cmpq %%rcx, %%rax\n")
	fmtPrintf("  %s %%al\n", inst)
	fmtPrintf("  movzbq %%al, %%rax\n") // true:1, false:0
	fmtPrintf("  pushq %%rax\n")
}

func emitStore(t *Type) {
	emitComment(2, "emitStore(%s)\n", kind(t))
	switch kind(t) {
	case T_SLICE:
		emitPopSlice()
		fmtPrintf("  popq %%rsi # lhs ptr addr\n")
		fmtPrintf("  movq %%rax, %d(%%rsi) # ptr to ptr\n", Itoa(0))
		fmtPrintf("  movq %%rcx, %d(%%rsi) # len to len\n", Itoa(8))
		fmtPrintf("  movq %%rdx, %d(%%rsi) # cap to cap\n", Itoa(16))
	case T_STRING:
		emitPopString()
		fmtPrintf("  popq %%rsi # lhs ptr addr\n")
		fmtPrintf("  movq %%rax, %d(%%rsi) # ptr to ptr\n", Itoa(0))
		fmtPrintf("  movq %%rcx, %d(%%rsi) # len to len\n", Itoa(8))
	case T_INT, T_BOOL, T_UINTPTR, T_POINTER:
		fmtPrintf("  popq %%rdi # rhs evaluated\n")
		fmtPrintf("  popq %%rax # lhs addr\n")
		fmtPrintf("  movq %%rdi, (%%rax) # assign\n")
	case T_UINT8:
		fmtPrintf("  popq %%rdi # rhs evaluated\n")
		fmtPrintf("  popq %%rax # lhs addr\n")
		fmtPrintf("  movb %%dil, (%%rax) # assign byte\n")
	case T_UINT16:
		fmtPrintf("  popq %%rdi # rhs evaluated\n")
		fmtPrintf("  popq %%rax # lhs addr\n")
		fmtPrintf("  movw %%di, (%%rax) # assign word\n")
	case T_STRUCT:
		// @FXIME
	case T_ARRAY:
		fmtPrintf("  popq %%rdi # rhs: addr of data\n")
		fmtPrintf("  popq %%rax # lhs: addr to store\n")
		fmtPrintf("  pushq $%d # size\n", Itoa(getSizeOfType(t)))
		fmtPrintf("  pushq %%rax # dst lhs\n")
		fmtPrintf("  pushq %%rdi # src rhs\n")
		fmtPrintf("  callq runtime.memcopy\n")
		emitRevertStackPointer(ptrSize*2 + intSize)
	default:
		panic2(__func__, "TBI:"+kind(t))
	}
}

func emitAssign(lhs *astExpr, rhs *astExpr) {
	emitComment(2, "Assignment: emitAddr(lhs:%s)\n", lhs.dtype)
	emitAddr(lhs)
	emitComment(2, "Assignment: emitExpr(rhs)\n")
	emitExpr(rhs, getTypeOfExpr(lhs))
	emitStore(getTypeOfExpr(lhs))
}

func emitStmt(stmt *astStmt) {
	emitComment(2, "\n")
	emitComment(2, "== Stmt %s ==\n", stmt.dtype)
	switch stmt.dtype {
	case "*astBlockStmt":
		var stmt2 *astStmt
		for _, stmt2 = range stmt.blockStmt.List {
			emitStmt(stmt2)
		}
	case "*astExprStmt":
		emitExpr(stmt.exprStmt.X, nil)
	case "*astDeclStmt":
		var decl *astDecl = stmt.DeclStmt.Decl
		if decl.dtype != "*astGenDecl" {
			panic2(__func__, "[*astDeclStmt] internal error")
		}
		var genDecl = decl.genDecl
		var valSpec = genDecl.Spec.valueSpec
		var t = e2t(valSpec.Type)
		var ident = valSpec.Name
		var lhs = new(astExpr)
		lhs.dtype = "*astIdent"
		lhs.ident = ident
		var rhs *astExpr
		if valSpec.Value == nil {
			emitComment(2, "lhs addresss\n")
			emitAddr(lhs)
			emitComment(2, "emitZeroValue for %s\n", t.e.dtype)
			emitZeroValue(t)
			emitComment(2, "Assignment: zero value\n")
			emitStore(t)
		} else {
			rhs = valSpec.Value
			emitAssign(lhs, rhs)
		}

		//var valueSpec *astValueSpec = genDecl.Specs[0]
		//var obj *astObject = valueSpec.Name.Obj
		//var typ *astExpr = valueSpec.Type
		//fmtPrintf("[emitStmt] TBI declSpec:%s\n", valueSpec.Name.Name)
		//os.Exit(1)

	case "*astAssignStmt":
		switch stmt.assignStmt.Tok {
		case "=":
			var lhs = stmt.assignStmt.Lhs
			var rhs = stmt.assignStmt.Rhs
			emitAssign(lhs[0], rhs[0])
		default:
			panic2(__func__, "TBI: assignment of "+stmt.assignStmt.Tok)
		}
	case "*astReturnStmt":
		if len(stmt.returnStmt.Results) == 0 {
			fmtPrintf("  leave\n")
			fmtPrintf("  ret\n")
		} else if len(stmt.returnStmt.Results) == 1 {
			emitExpr(stmt.returnStmt.Results[0], nil) // @TODO forceType should be fetched from func decl
			var knd = kind(getTypeOfExpr(stmt.returnStmt.Results[0]))
			switch knd {
			case T_BOOL, T_INT, T_UINTPTR, T_POINTER:
				fmtPrintf("  popq %%rax # return 64bit\n")
			case T_STRING:
				fmtPrintf("  popq %%rax # return string (ptr)\n")
				fmtPrintf("  popq %%rdi # return string (len)\n")
			case T_SLICE:
				fmtPrintf("  popq %%rax # return string (ptr)\n")
				fmtPrintf("  popq %%rdi # return string (len)\n")
				fmtPrintf("  popq %%rsi # return string (cap)\n")
			default:
				panic2(__func__, "[*astReturnStmt] TBI:"+knd)
			}
			fmtPrintf("  leave\n")
			fmtPrintf("  ret\n")
		} else if len(stmt.returnStmt.Results) == 3 {
			// Special treatment to return a slice
			emitExpr(stmt.returnStmt.Results[2], nil) // @FIXME
			emitExpr(stmt.returnStmt.Results[1], nil) // @FIXME
			emitExpr(stmt.returnStmt.Results[0], nil) // @FIXME
			fmtPrintf("  popq %%rax # return 64bit\n")
			fmtPrintf("  popq %%rdi # return 64bit\n")
			fmtPrintf("  popq %%rsi # return 64bit\n")
		} else {
			panic2(__func__, "[*astReturnStmt] TBI\n")
		}
	case "*astIfStmt":
		emitComment(2, "if\n")

		labelid++
		var labelEndif = ".L.endif." + Itoa(labelid)
		var labelElse = ".L.else." + Itoa(labelid)

		emitExpr(stmt.ifStmt.Cond, nil)
		emitPopBool("if condition")
		fmtPrintf("  cmpq $1, %%rax\n")
		var bodyStmt = new(astStmt)
		bodyStmt.dtype = "*astBlockStmt"
		bodyStmt.blockStmt = stmt.ifStmt.Body
		if stmt.ifStmt.Else != nil {
			fmtPrintf("  jne %s # jmp if false\n", labelElse)
			emitStmt(bodyStmt) // then
			fmtPrintf("  jmp %s\n", labelEndif)
			fmtPrintf("  %s:\n", labelElse)
			emitStmt(stmt.ifStmt.Else) // then
		} else {
			fmtPrintf("  jne %s # jmp if false\n", labelEndif)
			emitStmt(bodyStmt) // then
		}
		fmtPrintf("  %s:\n", labelEndif)
		emitComment(2, "end if\n")
	case "*astForStmt":
		labelid++
		var labelCond = ".L.for.cond." + Itoa(labelid)
		var labelPost = ".L.for.post." + Itoa(labelid)
		var labelExit = ".L.for.exit." + Itoa(labelid)
		//forStmt, ok := mapForNodeToFor[s]
		//assert(ok, "map value should exist")
		stmt.forStmt.labelPost = labelPost
		stmt.forStmt.labelExit = labelExit

		if stmt.forStmt.Init != nil {
			emitStmt(stmt.forStmt.Init)
		}

		fmtPrintf("  %s:\n", labelCond)
		if stmt.forStmt.Cond != nil {
			emitExpr(stmt.forStmt.Cond, nil)
			emitPopBool("for condition")
			fmtPrintf("  cmpq $1, %%rax\n")
			fmtPrintf("  jne %s # jmp if false\n", labelExit)
		}
		emitStmt(blockStmt2Stmt(stmt.forStmt.Body))
		fmtPrintf("  %s:\n", labelPost) // used for "continue"
		if stmt.forStmt.Post != nil {
			emitStmt(stmt.forStmt.Post)
		}
		fmtPrintf("  jmp %s\n", labelCond)
		fmtPrintf("  %s:\n", labelExit)
	case "*astRangeStmt": // only for array and slice
		labelid++
		var labelCond = ".L.range.cond." + Itoa(labelid)
		var labelPost = ".L.range.post." + Itoa(labelid)
		var labelExit = ".L.range.exit." + Itoa(labelid)

		stmt.rangeStmt.labelPost = labelPost
		stmt.rangeStmt.labelExit = labelExit
		// initialization: store len(rangeexpr)
		emitComment(2, "ForRange Initialization\n")

		emitComment(2, "  assign length to lenvar\n")
		// lenvar = len(s.X)
		emitVariableAddr(stmt.rangeStmt.lenvar)
		emitLen(stmt.rangeStmt.X)
		emitStore(tInt)

		emitComment(2, "  assign 0 to indexvar\n")
		// indexvar = 0
		emitVariableAddr(stmt.rangeStmt.indexvar)
		emitZeroValue(tInt)
		emitStore(tInt)

		// init key variable with 0
		if stmt.rangeStmt.Key != nil {
			assert(stmt.rangeStmt.Key.dtype == "*astIdent", "key expr should be an ident", __func__)
			var keyIdent = stmt.rangeStmt.Key.ident
			if keyIdent.Name != "_" {
				emitAddr(stmt.rangeStmt.Key) // lhs
				emitZeroValue(tInt)
				emitStore(tInt)
			}
		}

		// Condition
		// if (indexvar < lenvar) then
		//   execute body
		// else
		//   exit
		emitComment(2, "ForRange Condition\n")
		fmtPrintf("  %s:\n", labelCond)

		emitVariableAddr(stmt.rangeStmt.indexvar)
		emitLoad(tInt)
		emitVariableAddr(stmt.rangeStmt.lenvar)
		emitLoad(tInt)
		emitCompExpr("setl")
		emitPopBool(" indexvar < lenvar")
		fmtPrintf("  cmpq $1, %%rax\n")
		fmtPrintf("  jne %s # jmp if false\n", labelExit)

		emitComment(2, "assign list[indexvar] value variables\n")
		var elemType = getTypeOfExpr(stmt.rangeStmt.Value)
		emitAddr(stmt.rangeStmt.Value) // lhs

		emitVariableAddr(stmt.rangeStmt.indexvar)
		emitLoad(tInt) // index value
		emitListElementAddr(stmt.rangeStmt.X, elemType)

		emitLoad(elemType)
		emitStore(elemType)

		// Body
		emitComment(2, "ForRange Body\n")
		emitStmt(blockStmt2Stmt(stmt.rangeStmt.Body))

		// Post statement: Increment indexvar and go next
		emitComment(2, "ForRange Post statement\n")
		fmtPrintf("  %s:\n", labelPost)           // used for "continue"
		emitVariableAddr(stmt.rangeStmt.indexvar) // lhs
		emitVariableAddr(stmt.rangeStmt.indexvar) // rhs
		emitLoad(tInt)
		emitAddConst(1, "indexvar value ++")
		emitStore(tInt)

		if stmt.rangeStmt.Key != nil {
			assert(stmt.rangeStmt.Key.dtype == "*astIdent", "key expr should be an ident", __func__)
			var keyIdent = stmt.rangeStmt.Key.ident
			if keyIdent.Name != "_" {
				emitAddr(stmt.rangeStmt.Key)              // lhs
				emitVariableAddr(stmt.rangeStmt.indexvar) // rhs
				emitLoad(tInt)
				emitStore(tInt)
			}
		}

		fmtPrintf("  jmp %s\n", labelCond)

		fmtPrintf("  %s:\n", labelExit)

	case "*astIncDecStmt":
		var addValue int
		switch stmt.incDecStmt.Tok {
		case "++":
			addValue = 1
		case "--":
			addValue = -1
		default:
			panic2(__func__, "Unexpected Tok="+stmt.incDecStmt.Tok)
		}
		emitAddr(stmt.incDecStmt.X)
		emitExpr(stmt.incDecStmt.X, nil)
		emitAddConst(addValue, "rhs ++ or --")
		emitStore(getTypeOfExpr(stmt.incDecStmt.X))
	case "*astSwitchStmt":
		labelid++
		var labelEnd = fmtSprintf(".L.switch.%s.exit", []string{Itoa(labelid)})
		if stmt.switchStmt.Tag == nil {
			panic2(__func__, "Omitted tag is not supported yet")
		}
		emitExpr(stmt.switchStmt.Tag, nil)
		var condType = getTypeOfExpr(stmt.switchStmt.Tag)
		var cases = stmt.switchStmt.Body.List
		emitComment(2, "[DEBUG] cases len=%s\n", Itoa(len(cases)))
		var labels = make([]string, len(cases), len(cases))
		var defaultLabel string
		var i int
		var c *astStmt
		emitComment(2, "Start comparison with cases\n")
		for i, c = range cases {
			emitComment(2, "CASES idx=%s\n", Itoa(i))
			assert(c.dtype == "*astCaseClause", "should be *astCaseClause", __func__)
			var cc = c.caseClause
			labelid++
			var labelCase = ".L.case." + Itoa(labelid)
			labels[i] = labelCase
			if len(cc.List) == 0 { // @TODO implement slice nil comparison
				defaultLabel = labelCase
				continue
			}
			var e *astExpr
			for _, e = range cc.List {
				assert(getSizeOfType(condType) <= 8 || kind(condType) == T_STRING, "should be one register size or string", __func__)
				emitPushStackTop(condType, "switch expr")
				emitExpr(e, nil)
				emitCompEq(condType)
				emitPopBool(" of switch-case comparison")
				fmtPrintf("  cmpq $1, %%rax\n")
				fmtPrintf("  je %s # jump if match\n", labelCase)
			}
		}
		emitComment(2, "End comparison with cases\n")

		// if no case matches, then jump to
		if defaultLabel != "" {
			// default
			fmtPrintf("  jmp %s\n", defaultLabel)
		} else {
			// exit
			fmtPrintf("  jmp %s\n", labelEnd)
		}

		emitRevertStackTop(condType)
		for i, c = range cases {
			assert(c.dtype == "*astCaseClause", "should be *astCaseClause", __func__)
			var cc = c.caseClause
			fmtPrintf("%s:\n", labels[i])
			var _s *astStmt
			for _, _s = range cc.Body {
				emitStmt(_s)
			}
			fmtPrintf("  jmp %s\n", labelEnd)
		}
		fmtPrintf("%s:\n", labelEnd)
	case "*astBranchStmt":
		var containerFor = stmt.branchStmt.currentFor
		var labelToGo string
		switch stmt.branchStmt.Tok {
		case "continue":
			switch containerFor.dtype {
			case "*astForStmt":
				labelToGo = containerFor.forStmt.labelPost
			case "*astRangeStmt":
				labelToGo = containerFor.rangeStmt.labelPost
			default:
				panic2(__func__, "unexpected container dtype="+containerFor.dtype)
			}
			fmtPrintf("jmp %s # continue\n", labelToGo)
		case "break":
			switch containerFor.dtype {
			case "*astForStmt":
				labelToGo = containerFor.forStmt.labelExit
			case "*astRangeStmt":
				labelToGo = containerFor.rangeStmt.labelExit
			default:
				panic2(__func__, "unexpected container dtype="+containerFor.dtype)
			}
			fmtPrintf("jmp %s # break\n", labelToGo)
		default:
			panic2(__func__, "unexpected tok="+stmt.branchStmt.Tok)
		}
	default:
		panic2(__func__, "TBI:"+stmt.dtype)
	}
}

func blockStmt2Stmt(block *astBlockStmt) *astStmt {
	var stmt = new(astStmt)
	stmt.dtype = "*astBlockStmt"
	stmt.blockStmt = block
	return stmt
}

func emitRevertStackTop(t *Type) {
	fmtPrintf("  addq $%s, %%rsp # revert stack top\n", Itoa(getSizeOfType(t)))
}

var labelid int

func emitFuncDecl(pkgPrefix string, fnc *Func) {
	var localarea = fnc.localarea
	fmtPrintf("\n")
	fmtPrintf("%s.%s: # args %d, locals %d\n",
		pkgPrefix, fnc.name, Itoa(fnc.argsarea), Itoa(fnc.localarea))

	fmtPrintf("  pushq %%rbp\n")
	fmtPrintf("  movq %%rsp, %%rbp\n")
	if localarea != 0 {
		fmtPrintf("  subq $%d, %%rsp # local area\n", Itoa(-localarea))
	}

	if fnc.Body != nil {
		emitStmt(blockStmt2Stmt(fnc.Body))
	}

	fmtPrintf("  leave\n")
	fmtPrintf("  ret\n")
}

func emitGlobalVariable(name *astIdent, t *Type, val *astExpr) {
	var typeKind = kind(t)
	fmtPrintf("%s: # T %s\n", name.Name, typeKind)
	switch typeKind {
	case T_STRING:
		if val != nil && val.dtype == "*astBasicLit" {
			var sl = getStringLiteral(val.basicLit)
			fmtPrintf("  .quad %s\n", sl.label)
			fmtPrintf("  .quad %d\n", Itoa(sl.strlen))
		} else {
			fmtPrintf("  .quad 0\n")
			fmtPrintf("  .quad 0\n")
		}
	case T_POINTER:
		fmtPrintf("  .quad 0 # pointer \n") // @TODO
	case T_UINTPTR:
		fmtPrintf("  .quad 0\n")
	case T_BOOL:
		if val != nil {
			switch val.dtype {
			case "*astIdent":
				switch val.ident.Obj {
				case gTrue:
					fmtPrintf("  .quad 1 # bool true\n")
				case gFalse:
					fmtPrintf("  .quad 0 # bool false\n")
				default:
					panic2(__func__, "")
				}
			default:
				panic2(__func__, "")
			}
		} else {
			fmtPrintf("  .quad 0 # bool zero value\n")
		}
	case T_INT:
		fmtPrintf("  .quad 0\n")
	case T_UINT8:
		fmtPrintf("  .byte 0\n")
	case T_UINT16:
		fmtPrintf("  .word 0\n")
	case T_SLICE:
		fmtPrintf("  .quad 0 # ptr\n")
		fmtPrintf("  .quad 0 # len\n")
		fmtPrintf("  .quad 0 # cap\n")
	case T_ARRAY:
		if val != nil {
			panic2(__func__, "TBI")
		}
		if t.e.dtype != "*astArrayType" {
			panic2(__func__, "Unexpected type:"+t.e.dtype)
		}
		var arrayType = t.e.arrayType
		if arrayType.Len == nil {
			panic2(__func__, "global slice is not supported")
		}
		if arrayType.Len.dtype != "*astBasicLit" {
			panic2(__func__, "shoulbe basic literal")
		}
		var basicLit = arrayType.Len.basicLit
		if len(basicLit.Value) > 1 {
			panic2(__func__, "array length >= 10 is not supported yet.")
		}
		var length = evalInt(arrayType.Len)
		emitComment(0, "[emitGlobalVariable] array length uint8=%s\n", Itoa(length))
		var zeroValue string
		var kind string = kind(e2t(arrayType.Elt))
		switch kind {
		case T_INT:
			zeroValue = "  .quad 0 # int zero value\n"
		case T_UINT8:
			zeroValue = "  .byte 0 # uint8 zero value\n"
		case T_STRING:
			zeroValue = "  .quad 0 # string zero value (ptr)\n"
			zeroValue = zeroValue + "  .quad 0 # string zero value (len)\n"
		default:
			panic2(__func__, "Unexpected kind:"+kind)
		}

		var i int
		for i = 0; i < length; i++ {
			fmtPrintf(zeroValue)
		}
	default:
		panic2(__func__, "TBI:kind="+typeKind)
	}
}

func emitData(pkgName string, vars []*astValueSpec, sliterals []*stringLiteralsContainer) {
	fmtPrintf(".data\n")
	emitComment(0, "string literals len = %s\n", Itoa(len(sliterals)))
	var con *stringLiteralsContainer
	for _, con = range sliterals {
		emitComment(0, "string literals\n")
		fmtPrintf("%s:\n", con.sl.label)
		fmtPrintf("  .string %s\n", con.sl.value)
	}

	emitComment(0, "===== Global Variables =====\n")

	var spec *astValueSpec
	for _, spec = range vars {
		var t *Type
		if spec.Type != nil {
			t = e2t(spec.Type)
		}
		emitGlobalVariable(spec.Name, t, spec.Value)
	}

	emitComment(0, "==============================\n")
}

func emitText(pkgName string, funcs []*Func) {
	fmtPrintf(".text\n")
	var fnc *Func
	for _, fnc = range funcs {
		emitFuncDecl(pkgName, fnc)
	}
}

func generateCode(pkgContainer *PkgContainer) {
	emitData(pkgContainer.name, pkgContainer.vars, stringLiterals)
	emitText(pkgContainer.name, pkgContainer.funcs)
}
