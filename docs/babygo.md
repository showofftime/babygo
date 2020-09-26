babygo是一个go语言编译器实现，算是一个介绍go编译器开发的项目吧。

- babygo支持的编程语言文法，就是go语言的文法；
- babygo中没有依赖其他第三方的库，使用的主要是go语言提供的标准库、系统调用实现；
- babygo中的词法分析器lexer、语法分析器parser、代码生成器等都是自己重写的，但是肯定借鉴了go中的实现；
- babygo编译输出的是一个静态二进制文件，和go编译器一样的，问题就是可能会二进制文件比较大；

foundation.go:
这个文件中实现了panic、panic2、throw、assert几个比较常用的函数，注意为什么不用go自带的panic实现呢，go中自带的panic会打印错误堆栈然后允许被recover，babygo中不是，它打印错误信息并退出。

libs.go:
这个文件中实现了fmtSprintf、fmtPrintf、logf等常见的打印操作，还有字符串、数值类型转换Itoa、Atoi等操作。


