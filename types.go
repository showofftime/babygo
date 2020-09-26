package main

// --- type ---
const sliceSize int = 24
const stringSize int = 16
const intSize int = 8
const ptrSize int = 8

type Type struct {
	//kind string
	e *astExpr
}

const T_STRING string = "T_STRING"
const T_SLICE string = "T_SLICE"
const T_BOOL string = "T_BOOL"
const T_INT string = "T_INT"
const T_UINT8 string = "T_UINT8"
const T_UINT16 string = "T_UINT16"
const T_UINTPTR string = "T_UINTPTR"
const T_ARRAY string = "T_ARRAY"
const T_STRUCT string = "T_STRUCT"
const T_POINTER string = "T_POINTER"

var tInt *Type
var tUint8 *Type
var tUint16 *Type
var tUintptr *Type
var tString *Type
var tBool *Type

func getTypeOfExpr(expr *astExpr) *Type {
	//emitComment(0, "[%s] start\n", __func__)
	switch expr.dtype {
	case "*astIdent":
		switch expr.ident.Obj.Kind {
		case astVar:
			switch expr.ident.Obj.Decl.dtype {
			case "*astValueSpec":
				var decl = expr.ident.Obj.Decl.valueSpec
				var t = new(Type)
				t.e = decl.Type
				return t
			case "*astField":
				var decl = expr.ident.Obj.Decl.field
				var t = new(Type)
				t.e = decl.Type
				return t
			default:
				panic2(__func__, "ERROR 0\n")
			}
		case astConst:
			switch expr.ident.Obj {
			case gTrue, gFalse:
				return tBool
			}
			switch expr.ident.Obj.Decl.dtype {
			case "*astValueSpec":
				return e2t(expr.ident.Obj.Decl.valueSpec.Type)
			default:
				panic2(__func__, "cannot decide type of cont ="+expr.ident.Obj.Name)
			}
		default:
			panic2(__func__, "2:Obj.Kind="+expr.ident.Obj.Kind)
		}
	case "*astBasicLit":
		switch expr.basicLit.Kind {
		case "STRING":
			return tString
		case "INT":
			return tInt
		case "CHAR":
			return tInt
		default:
			panic2(__func__, "TBI:"+expr.basicLit.Kind)
		}
	case "*astIndexExpr":
		var list = expr.indexExpr.X
		return getElementTypeOfListType(getTypeOfExpr(list))
	case "*astUnaryExpr":
		switch expr.unaryExpr.Op {
		case "-":
			return getTypeOfExpr(expr.unaryExpr.X)
		case "!":
			return tBool
		default:
			panic2(__func__, "TBI: Op="+expr.unaryExpr.Op)
		}
	case "*astCallExpr":
		emitComment(0, "[%s] *astCallExpr\n", __func__)
		var fun = expr.callExpr.Fun
		switch fun.dtype {
		case "*astIdent":
			var fn = fun.ident
			if fn.Obj == nil {
				panic2(__func__, "[astCallExpr] nil Obj is not allowed")
			}
			switch fn.Obj.Kind {
			case astType:
				return e2t(fun)
			case astFunc:
				switch fn.Obj {
				case gLen, gCap:
					return tInt
				case gNew:
					var starExpr = new(astStarExpr)
					starExpr.X = expr.callExpr.Args[0]
					var eStarExpr = new(astExpr)
					eStarExpr.dtype = "*astStarExpr"
					eStarExpr.starExpr = starExpr
					return e2t(eStarExpr)
				case gMake:
					return e2t(expr.callExpr.Args[0])
				}
				var decl = fn.Obj.Decl
				if decl == nil {
					panic2(__func__, "decl of function "+fn.Name+" is  nil")
				}
				switch decl.dtype {
				case "*astFuncDecl":
					var resultList = decl.funcDecl.Type.results.List
					if len(resultList) != 1 {
						panic2(__func__, "[astCallExpr] len results.List is not 1")
					}
					return e2t(decl.funcDecl.Type.results.List[0].Type)
				default:
					panic2(__func__, "[astCallExpr] decl.dtype="+decl.dtype)
				}
				panic2(__func__, "[astCallExpr] Fun ident "+fn.Name)
			}
		case "*astArrayType":
			return e2t(fun)
		default:
			panic2(__func__, "[astCallExpr] dtype="+expr.callExpr.Fun.dtype)
		}
	case "*astSliceExpr":
		var underlyingCollectionType = getTypeOfExpr(expr.sliceExpr.X)
		var elementTyp *astExpr
		switch underlyingCollectionType.e.dtype {
		case "*astArrayType":
			elementTyp = underlyingCollectionType.e.arrayType.Elt
		}
		var t = new(astArrayType)
		t.Len = nil
		t.Elt = elementTyp
		var e = new(astExpr)
		e.dtype = "*astArrayType"
		e.arrayType = t
		return e2t(e)
	case "*astStarExpr":
		var t = getTypeOfExpr(expr.starExpr.X)
		var ptrType = t.e.starExpr
		if ptrType == nil {
			panic2(__func__, "starExpr shoud not be nil")
		}
		return e2t(ptrType.X)
	case "*astBinaryExpr":
		switch expr.binaryExpr.Op {
		case "==", "!=", "<", ">", "<=", ">=":
			return tBool
		default:
			return getTypeOfExpr(expr.binaryExpr.X)
		}
	case "*astSelectorExpr":
		var structType = getStructTypeOfX(expr.selectorExpr)
		var field = lookupStructField(getStructTypeSpec(structType), expr.selectorExpr.Sel.Name)
		return e2t(field.Type)
	case "*astCompositeLit":
		return e2t(expr.compositeLit.Type)
	case "*astParenExpr":
		return getTypeOfExpr(expr.parenExpr.X)
	default:
		panic2(__func__, "TBI:dtype="+expr.dtype)
	}

	panic2(__func__, "nil type is not allowed\n")
	var r *Type
	return r
}

func e2t(typeExpr *astExpr) *Type {
	if typeExpr == nil {
		panic2(__func__, "nil is not allowed")
	}
	var r = new(Type)
	r.e = typeExpr
	return r
}

func kind(t *Type) string {
	if t == nil {
		panic2(__func__, "nil type is not expected\n")
	}
	var e = t.e
	switch t.e.dtype {
	case "*astIdent":
		var ident = t.e.ident
		switch ident.Name {
		case "uintptr":
			return T_UINTPTR
		case "int":
			return T_INT
		case "string":
			return T_STRING
		case "uint8":
			return T_UINT8
		case "uint16":
			return T_UINT16
		case "bool":
			return T_BOOL
		default:
			// named type
			var decl = ident.Obj.Decl
			if decl.dtype != "*astTypeSpec" {
				panic2(__func__, "unsupported decl :"+decl.dtype)
			}
			var typeSpec = decl.typeSpec
			return kind(e2t(typeSpec.Type))
		}
	case "*astStructType":
		return T_STRUCT
	case "*astArrayType":
		if e.arrayType.Len == nil {
			return T_SLICE
		} else {
			return T_ARRAY
		}
	case "*astStarExpr":
		return T_POINTER
	case "*astEllipsis": // x ...T
		return T_SLICE // @TODO is this right ?
	default:
		panic2(__func__, "Unkown dtype:"+t.e.dtype)
	}
	panic2(__func__, "error")
	return ""
}

func getStructTypeOfX(e *astSelectorExpr) *Type {
	var typeOfX = getTypeOfExpr(e.X)
	var structType *Type
	switch kind(typeOfX) {
	case T_STRUCT:
		// strct.field => e.X . e.Sel
		structType = typeOfX
	case T_POINTER:
		// ptr.field => e.X . e.Sel
		assert(typeOfX.e.dtype == "*astStarExpr", "should be astStarExpr", __func__)
		var ptrType = typeOfX.e.starExpr
		structType = e2t(ptrType.X)
	default:
		panic2(__func__, "TBI")
	}
	return structType
}

func getElementTypeOfListType(t *Type) *Type {
	switch kind(t) {
	case T_SLICE, T_ARRAY:
		var arrayType = t.e.arrayType
		if arrayType == nil {
			panic2(__func__, "should not be nil")
		}
		return e2t(arrayType.Elt)
	case T_STRING:
		return tUint8
	default:
		panic2(__func__, "TBI kind="+kind(t))
	}
	var r *Type
	return r
}

func getSizeOfType(t *Type) int {
	var knd = kind(t)
	switch kind(t) {
	case T_SLICE:
		return sliceSize
	case T_STRING:
		return 16
	case T_ARRAY:
		var elemSize = getSizeOfType(e2t(t.e.arrayType.Elt))
		return elemSize * evalInt(t.e.arrayType.Len)
	case T_INT, T_UINTPTR, T_POINTER:
		return 8
	case T_UINT8:
		return 1
	case T_BOOL:
		return 8
	case T_STRUCT:
		return calcStructSizeAndSetFieldOffset(getStructTypeSpec(t))
	default:
		panic2(__func__, "TBI:"+knd)
	}
	return 0
}

func getPushSizeOfType(t *Type) int {
	switch kind(t) {
	case T_SLICE:
		return sliceSize
	case T_STRING:
		return stringSize
	case T_UINT8, T_UINT16, T_INT, T_BOOL:
		return intSize
	case T_UINTPTR, T_POINTER:
		return ptrSize
	case T_ARRAY, T_STRUCT:
		return ptrSize
	default:
		throw(kind(t))
	}
	throw(kind(t))
	return 0
}

func getStructFieldOffset(field *astField) int {
	var offset = field.Offset
	return offset
}

func setStructFieldOffset(field *astField, offset int) {
	field.Offset = offset
}

func getStructFields(structTypeSpec *astTypeSpec) []*astField {
	if structTypeSpec.Type.dtype != "*astStructType" {
		panic2(__func__, "Unexpected dtype")
	}
	var structType = structTypeSpec.Type.structType
	return structType.Fields.List
}

func getStructTypeSpec(namedStructType *Type) *astTypeSpec {
	if kind(namedStructType) != T_STRUCT {
		panic2(__func__, "not T_STRUCT")
	}
	if namedStructType.e.dtype != "*astIdent" {
		panic2(__func__, "not ident")
	}

	var ident = namedStructType.e.ident

	if ident.Obj.Decl.dtype != "*astTypeSpec" {
		panic2(__func__, "not *astTypeSpec")
	}

	var typeSpec = ident.Obj.Decl.typeSpec
	return typeSpec
}

func lookupStructField(structTypeSpec *astTypeSpec, selName string) *astField {
	var field *astField
	for _, field = range getStructFields(structTypeSpec) {
		if field.Name.Name == selName {
			return field
		}
	}
	panic("Unexpected flow: struct field not found:" + selName)
	return field
}

func calcStructSizeAndSetFieldOffset(structTypeSpec *astTypeSpec) int {
	var offset int = 0

	var fields = getStructFields(structTypeSpec)
	var field *astField
	for _, field = range fields {
		setStructFieldOffset(field, offset)
		var size = getSizeOfType(e2t(field.Type))
		offset = offset + size
	}
	return offset
}

