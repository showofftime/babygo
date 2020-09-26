package main

// scanner就是babygo的词法分析器（也常称为lexer）

var (
	scannerSrc	[]uint8 // scanner扫描的源码
	scannerCh	uint8 	// scanner扫描出的字符

	scannerOffset		int // scanner当前扫描位置
	scannerNextOffset	int // scanner下一个扫描位置

	scannerInsertSemi bool	// scanner当前扫描位置是否要插入一个分号;
)

// scannerNext 扫描下一个字符，扫描出的字符保存在全局变量scannerCh中，如果为1表示已经扫描结束
func scannerNext() {
	if scannerNextOffset < len(scannerSrc) {
		scannerOffset = scannerNextOffset
		scannerCh = scannerSrc[scannerOffset]
		scannerNextOffset++
	} else {
		scannerOffset = len(scannerSrc)
		scannerCh = 1 //EOF
	}
}

// keywords go语言定义的关键字, see https://golang.org/ref/spec#Keywords
var keywords = []string{
	"break", "default", "func", "interface", "select",
	"case", "defer", "go", "map", "struct",
	"chan", "else", "goto", "package", "switch",
	"const", "fallthrough", "if", "range", "type",
	"continue", "for", "import", "return", "var",
}

// scannerInit 初始化scanner
func scannerInit(src []uint8) {
	scannerSrc = src
	scannerOffset = 0
	scannerNextOffset = 0
	scannerInsertSemi = false
	scannerCh = ' '
	logf("src len = %s\n", Itoa(len(scannerSrc)))
	scannerNext()
}

// isLetter 判断字符ch是否是一个英文字母
func isLetter(ch uint8) bool {
	if ch == '_' {
		return true
	}
	return ('A' <= ch && ch <= 'Z') || ('a' <= ch && ch <= 'z')
}

// isDecimal 判断字符ch是否是一个数值
func isDecimal(ch uint8) bool {
	return '0' <= ch && ch <= '9'
}

// scannerScanIdentifier 遇到一个标识符开始，让scanner持续扫描字符，直到扫描到一个identifier（标识符）
//
// Note: 其实标识符的定义规则可以概括成：
// - 由字母、数字、下划线构成；
// - 首字母由字母或者下划线构成；
//
// 所以应该知道babygo中的实现是简化版的，并没有考虑那么周全
func scannerScanIdentifier() string {
	var offset = scannerOffset
	// TODO check if first letter is letter or understore character
	// TODO check if scanned char is letter or understore or digit
	for isLetter(scannerCh) || isDecimal(scannerCh) {
		scannerNext()
	}
	return string(scannerSrc[offset:scannerOffset])
}

// scannerScanNumber 遇到一个数字，让scanner持续扫描字符，直到扫描到一个完整的number
//
// number不只是有整数，还有浮点数，浮点数也有各种不同的书写形式，如科学计数法等，所以这里的扫描number
// 也是一个简化的版本
func scannerScanNumber() string {
	var offset = scannerOffset
	for isDecimal(scannerCh) {
		scannerNext()
	}
	return string(scannerSrc[offset:scannerOffset])
}

// scannerScanString 遇到"，让scanner持续扫描字符，直到扫描到一个完整的string
//
// go语言定义中其实``也可以用来定义字符串，所以这里也是一个简化的版本
func scannerScanString() string {
	var (
		offset = scannerOffset - 1
		escaped bool
	)
	for !escaped && scannerCh != '"' {
		if scannerCh == '\\' {
			escaped = true
			scannerNext()
			scannerNext()
			escaped = false
			continue
		}
		scannerNext()
	}
	scannerNext() // consume ending '""
	return string(scannerSrc[offset:scannerOffset])
}

// scannerScanChar 遇到'，让scanner扫描一个字符出来
func scannerScanChar() string {
	// '\'' opening already consumed
	var offset = scannerOffset - 1
	var ch uint8
	for {
		ch = scannerCh
		scannerNext()
		if ch == '\'' {
			break
		}
		// 转义字符
		if ch == '\\' {
			scannerNext()
		}
	}

	return string(scannerSrc[offset:scannerOffset])
}

// scannerScanComment 遇到一个行注释的开始，扫描一个完整的注释行
//
// go里面也可以使用块注释/* */，这里也是一个简化的版本
func scannerrScanComment() string {
	var offset = scannerOffset - 1
	for scannerCh != '\n' {
		scannerNext()
	}
	return string(scannerSrc[offset:scannerOffset])
}

// TokenContainer Token容器类型，标识了token出现的位置、token类型、字面量
//
// 关于token的详细说明，可以参考go官方的文档：https://golang.org/ref/spec#Tokens
type TokenContainer struct {
	pos int    // TODO what's this ?
	tok string // token.Token
	lit string // raw data
}

// scannerSkipWhitespace 扫描时跳过空白字符
func scannerSkipWhitespace() {
	for scannerCh == ' ' || scannerCh == '\t' || (scannerCh == '\n' && !scannerInsertSemi) || scannerCh == '\r' {
		scannerNext()
	}
}

// scannerScan 让scanner扫描，并将扫描出的对象作为一个TokenContainer返回
//
// 这里其实是一个状态机，每遇到一个token起始字符时，就让scanner持续扫描，直到完成一个完整的token扫描或出错，
// 成功扫描出的字面量，如标识符、字符串、数字等等，将被封装成一个TokenContainer返回
func scannerScan() *TokenContainer {

	scannerSkipWhitespace()

	var (
		tc = new(TokenContainer)
		lit string
		tok string
		insertSemi bool

		ch = scannerCh	// token字面量的起始字符
	)
	
	if isLetter(ch) {
		// 如果首字符是一个字母，接下来应该扫描一个完整的identifier
		lit = scannerScanIdentifier()
		if inArray(lit, keywords) {
			// 此token是一个keyword
			tok = lit
			switch tok {
			case "break", "continue", "fallthrough", "return":
				insertSemi = true
			}
		} else {
			// 此token是一个identifier
			insertSemi = true
			tok = "IDENT"
		}
	} else if isDecimal(ch) {
		// 如果首字符是一个digit，接下来应该继续扫描一个完整的number
		insertSemi = true
		lit = scannerScanNumber()
		tok = "INT"
	} else {
		// 不是标识符，也不是数值，那是什么呢？接下来可能是字符、字符串、空白字符等等
		scannerNext()
		switch ch {
		case '\n':
			// 换行符
			tok = ";"
			lit = "\n"
			insertSemi = false
		case '"': // double quote
			// 字符串起始字符"
			insertSemi = true
			lit = scannerScanString()
			tok = "STRING"
		case '\'': // single quote
			// 字符起始字符'
			insertSemi = true
			lit = scannerScanChar()
			tok = "CHAR"
		// https://golang.org/ref/spec#Operators_and_punctuation
		//	+    &     +=    &=     &&    ==    !=    (    )
		//	-    |     -=    |=     ||    <     <=    [    ]
		//  *    ^     *=    ^=     <-    >     >=    {    }
		//	/    <<    /=    <<=    ++    =     :=    ,    ;
		//	%    >>    %=    >>=    --    !     ...   .    :
		//	&^          &^=
		case ':': // :=, :
			// := 或者 :，前者是短变量赋值，后者是普通的struct field初始化或者map kv初始化
			if scannerCh == '=' {
				scannerNext()
				tok = ":="
			} else {
				tok = ":"
			}
		case '.': // ..., .
			// 首字符是.，那么接下来可能是...变长数组
			var peekCh = scannerSrc[scannerNextOffset]
			if scannerCh == '.' && peekCh == '.' {
				scannerNext()
				scannerNext()
				tok = "..."
			} else {
				// 也有可能就是普通的.了，.有什么用呢，struct字段访问
				tok = "."
			}
		case ',':
			// 首字符是,，这种接下来可能是什么呢？很多
			tok = ","
		case ';':
			// 首字符是;，基本上是一条语句结束
			tok = ";"
			lit = ";"
		case '(':
			// 首字符是,，表示左括号开始，内部可能是expr，也可能是参数列表等等
			tok = "("
		case ')':
			// 字符是)，标识了一个()的结束
			insertSemi = true
			tok = ")"
		case '[':
			tok = "["
		case ']':
			insertSemi = true
			tok = "]"
		case '{':
			tok = "{"
		case '}':
			insertSemi = true
			tok = "}"
		case '+': // +=, ++, +
			// 运算符+=、++，或者+
			switch scannerCh {
			case '=':
				scannerNext()
				tok = "+="
			case '+':
				scannerNext()
				tok = "++"
				insertSemi = true
			default:
				tok = "+"
			}
		case '-': // -= --  -
			// 运算符-=，--，或者-
			switch scannerCh {
			case '-':
				scannerNext()
				tok = "--"
				insertSemi = true
			case '=':
				scannerNext()
				tok = "-="
			default:
				tok = "-"
			}
		case '*': // *=  *
			// 运算符*=, *
			if scannerCh == '=' {
				scannerNext()
				tok = "*="
			} else {
				tok = "*"
			}
		case '/':
			// 首字符是/，接下来可能是//行注释，也可能是/* */块注释，也可能是/=, /运算符
			if scannerCh == '/' {
				// comment
				// @TODO block comment
				if scannerInsertSemi {
					scannerCh = '/'
					scannerOffset = scannerOffset - 1
					scannerNextOffset = scannerOffset + 1
					tc.lit = "\n"
					tc.tok = ";"
					scannerInsertSemi = false
					return tc
				}
				lit = scannerrScanComment()
				tok = "COMMENT"
			} else if scannerCh == '=' {
				tok = "/="
			} else {
				tok = "/"
			}
		case '%': // %= %
			if scannerCh == '=' {
				scannerNext()
				tok = "%="
			} else {
				tok = "%"
			}
		case '^': // ^= ^
			if scannerCh == '=' {
				scannerNext()
				tok = "^="
			} else {
				tok = "^"
			}
		case '<': //  <= <- <<= <<
			switch scannerCh {
			case '-':
				scannerNext()
				tok = "<-"
			case '=':
				scannerNext()
				tok = "<="
			case '<':
				var peekCh = scannerSrc[scannerNextOffset]
				if peekCh == '=' {
					scannerNext()
					scannerNext()
					tok = "<<="
				} else {
					scannerNext()
					tok = "<<"
				}
			default:
				tok = "<"
			}
		case '>': // >= >>= >> >
			switch scannerCh {
			case '=':
				scannerNext()
				tok = ">="
			case '>':
				var peekCh = scannerSrc[scannerNextOffset]
				if peekCh == '=' {
					scannerNext()
					scannerNext()
					tok = ">>="
				} else {
					scannerNext()
					tok = ">>"
				}
			default:
				tok = ">"
			}
		case '=': // == =
			if scannerCh == '=' {
				scannerNext()
				tok = "=="
			} else {
				tok = "="
			}
		case '!': // !=, !
			if scannerCh == '=' {
				scannerNext()
				tok = "!="
			} else {
				tok = "!"
			}
		case '&': // & &= && &^ &^=
			switch scannerCh {
			case '=':
				scannerNext()
				tok = "&="
			case '&':
				scannerNext()
				tok = "&&"
			case '^':
				var peekCh = scannerSrc[scannerNextOffset]
				if peekCh == '=' {
					scannerNext()
					scannerNext()
					tok = "&^="
				} else {
					scannerNext()
					tok = "&^"
				}
			default:
				tok = "&"
			}
		case '|': // |= || |
			switch scannerCh {
			case '|':
				scannerNext()
				tok = "||"
			case '=':
				scannerNext()
				tok = "|="
			default:
				tok = "|"
			}
		case 1:
			// 如果是1，则表示扫描到了EOF，结束了，通常我们使用-1表示io.EOF，不知道作者这里为什么使用1来表示，
			// anyway！可能是因为uint8的原因吧，这里任何字符<48的字符都可以，只要能帮助我们区分普通字符就可以了
			tok = "EOF"
		default:
			panic2(__func__, "unknown char:"+string([]uint8{ch})+":"+Itoa(int(ch)))
			tok = "UNKNOWN"
		}
	}
	tc.lit = lit
	tc.pos = 0 // why is zero?
	tc.tok = tok
	scannerInsertSemi = insertSemi
	return tc
}
