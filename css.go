package main

const defaultCSS = ` /* Pimbin's default CSS (Solarized Dark)*/

body { font-family: monospace; color: #93a1a1; background-color: #002b36 }

a, a:visited {color: #93a1a1;}

.filename, body > h1 {font-size: 1.25em;}
.filename {display: inline;}

#file-index {list-style-type: none; padding-left: 2ch;}

/* Background */ .chroma { color: #93a1a1; background-color: #002b36 }
/* Other */ .chroma .x { color: #cb4b16 }
/* LineTableTD */ .chroma .lntd { vertical-align: top; padding: 0; margin: 0; border: 0; }
/* LineTable */ .chroma .lntable { border-spacing: 0; padding: 0; margin: 0; border: 0; width: auto; overflow: auto; display: block; }
/* LineHighlight */ .chroma .hl { display: block; width: 100%;background-color: #19404a }
/* LineNumbersTable */ .chroma .lnt { margin-right: 0.4em; padding: 0 0.4em 0 0.4em;color: #495050 }
/* LineNumbers */ .chroma .ln { margin-right: 0.4em; padding: 0 0.4em 0 0.4em;color: #495050 }
/* Keyword */ .chroma .k { color: #719e07 }
/* KeywordConstant */ .chroma .kc { color: #cb4b16 }
/* KeywordDeclaration */ .chroma .kd { color: #268bd2 }
/* KeywordNamespace */ .chroma .kn { color: #719e07 }
/* KeywordPseudo */ .chroma .kp { color: #719e07 }
/* KeywordReserved */ .chroma .kr { color: #268bd2 }
/* KeywordType */ .chroma .kt { color: #dc322f }
/* NameBuiltin */ .chroma .nb { color: #b58900 }
/* NameBuiltinPseudo */ .chroma .bp { color: #268bd2 }
/* NameClass */ .chroma .nc { color: #268bd2 }
/* NameConstant */ .chroma .no { color: #cb4b16 }
/* NameDecorator */ .chroma .nd { color: #268bd2 }
/* NameEntity */ .chroma .ni { color: #cb4b16 }
/* NameException */ .chroma .ne { color: #cb4b16 }
/* NameFunction */ .chroma .nf { color: #268bd2 }
/* NameTag */ .chroma .nt { color: #268bd2 }
/* NameVariable */ .chroma .nv { color: #268bd2 }
/* LiteralString */ .chroma .s { color: #2aa198 }
/* LiteralStringAffix */ .chroma .sa { color: #2aa198 }
/* LiteralStringBacktick */ .chroma .sb { color: #586e75 }
/* LiteralStringChar */ .chroma .sc { color: #2aa198 }
/* LiteralStringDelimiter */ .chroma .dl { color: #2aa198 }
/* LiteralStringDouble */ .chroma .s2 { color: #2aa198 }
/* LiteralStringEscape */ .chroma .se { color: #cb4b16 }
/* LiteralStringInterpol */ .chroma .si { color: #2aa198 }
/* LiteralStringOther */ .chroma .sx { color: #2aa198 }
/* LiteralStringRegex */ .chroma .sr { color: #dc322f }
/* LiteralStringSingle */ .chroma .s1 { color: #2aa198 }
/* LiteralStringSymbol */ .chroma .ss { color: #2aa198 }
/* LiteralNumber */ .chroma .m { color: #2aa198 }
/* LiteralNumberBin */ .chroma .mb { color: #2aa198 }
/* LiteralNumberFloat */ .chroma .mf { color: #2aa198 }
/* LiteralNumberHex */ .chroma .mh { color: #2aa198 }
/* LiteralNumberInteger */ .chroma .mi { color: #2aa198 }
/* LiteralNumberIntegerLong */ .chroma .il { color: #2aa198 }
/* LiteralNumberOct */ .chroma .mo { color: #2aa198 }
/* Operator */ .chroma .o { color: #719e07 }
/* OperatorWord */ .chroma .ow { color: #719e07 }
/* Comment */ .chroma .c { color: #586e75 }
/* CommentHashbang */ .chroma .ch { color: #586e75 }
/* CommentMultiline */ .chroma .cm { color: #586e75 }
/* CommentSingle */ .chroma .c1 { color: #586e75 }
/* CommentSpecial */ .chroma .cs { color: #719e07 }
/* CommentPreproc */ .chroma .cp { color: #719e07 }
/* CommentPreprocFile */ .chroma .cpf { color: #719e07 }
/* GenericDeleted */ .chroma .gd { color: #dc322f }
/* GenericEmph */ .chroma .ge { font-style: italic }
/* GenericError */ .chroma .gr { color: #dc322f; font-weight: bold }
/* GenericHeading */ .chroma .gh { color: #cb4b16 }
/* GenericInserted */ .chroma .gi { color: #719e07 }
/* GenericStrong */ .chroma .gs { font-weight: bold }
/* GenericSubheading */ .chroma .gu { color: #268bd2 }
`