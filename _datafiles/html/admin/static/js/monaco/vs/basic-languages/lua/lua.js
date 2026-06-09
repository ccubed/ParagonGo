/*!-----------------------------------------------------------------------------
 * Copyright (c) Microsoft Corporation. All rights reserved.
 * Version: 0.52.2(404545bded1df6ffa41ea0af4e8ddb219018c6c1)
 * Released under the MIT license
 * https://github.com/microsoft/monaco-editor/blob/main/LICENSE.txt
 *-----------------------------------------------------------------------------*/

// Lua basic-language definition.
//
// The vendored Monaco distribution only ships the javascript and typescript
// grammars. This module supplies the canonical upstream monaco-editor Lua
// grammar in the same AMD module shape the other basic-language files use,
// exporting `conf` (language configuration) and `language` (Monarch tokens).
//
// Unlike the upstream build, this distribution has no basic-languages
// contribution manifest, so registration is performed explicitly by
// monaco-editor-frame.js (it requires this module and calls
// monaco.languages.register + setMonarchTokensProvider).
define("vs/basic-languages/lua/lua", ["require"], function () {
    "use strict";

    var conf = {
        comments: {
            lineComment: "--",
            blockComment: ["--[[", "]]"]
        },
        brackets: [
            ["{", "}"],
            ["[", "]"],
            ["(", ")"]
        ],
        autoClosingPairs: [
            { open: "{", close: "}" },
            { open: "[", close: "]" },
            { open: "(", close: ")" },
            { open: '"', close: '"' },
            { open: "'", close: "'" }
        ],
        surroundingPairs: [
            { open: "{", close: "}" },
            { open: "[", close: "]" },
            { open: "(", close: ")" },
            { open: '"', close: '"' },
            { open: "'", close: "'" }
        ]
    };

    var language = {
        defaultToken: "",
        tokenPostfix: ".lua",
        keywords: [
            "and", "break", "do", "else", "elseif", "end", "false", "for",
            "function", "goto", "if", "in", "local", "nil", "not", "or",
            "repeat", "return", "then", "true", "until", "while"
        ],
        brackets: [
            { token: "delimiter.bracket", open: "{", close: "}" },
            { token: "delimiter.array", open: "[", close: "]" },
            { token: "delimiter.parenthesis", open: "(", close: ")" }
        ],
        operators: [
            "+", "-", "*", "/", "%", "^", "#", "==", "~=", "<=", ">=", "<",
            ">", "=", ";", ":", ",", ".", "..", "..."
        ],
        symbols: /[=><!~?:&|+\-*\/\^%#]+/,
        escapes: /\\(?:[abfnrtv\\"']|x[0-9A-Fa-f]{1,2}|[0-7]{1,3}|z\s*)/,
        tokenizer: {
            root: [
                [
                    /[a-zA-Z_]\w*/,
                    {
                        cases: {
                            "@keywords": { token: "keyword.$0" },
                            "@default": "identifier"
                        }
                    }
                ],
                { include: "@whitespace" },
                [/(,)(\s*)([a-zA-Z_]\w*)(\s*)(:)(?!:)/, ["delimiter", "", "key", "", "delimiter"]],
                [/({)(\s*)([a-zA-Z_]\w*)(\s*)(:)(?!:)/, ["@brackets", "", "key", "", "delimiter"]],
                [/[{}()\[\]]/, "@brackets"],
                [/@symbols/, { cases: { "@operators": "delimiter", "@default": "" } }],
                [/\d*\.\d+([eE][\-+]?\d+)?/, "number.float"],
                [/0[xX][0-9a-fA-F]+/, "number.hex"],
                [/\d+/, "number"],
                [/[;,.]/, "delimiter"],
                [/"([^"\\]|\\.)*$/, "string.invalid"],
                [/'([^'\\]|\\.)*$/, "string.invalid"],
                [/"/, "string", '@string."'],
                [/'/, "string", "@string.'"],
                [/\[(=*)\[/, "string", "@bracketedString.$1"]
            ],
            whitespace: [
                [/[ \t\r\n]+/, ""],
                [/--\[(=*)\[/, "comment", "@comment.$1"],
                [/--.*$/, "comment"]
            ],
            comment: [
                [/[^\]]+/, "comment"],
                [
                    /\](=*)\]/,
                    {
                        cases: {
                            "$1==$S2": { token: "comment", next: "@pop" },
                            "@default": "comment"
                        }
                    }
                ],
                [/./, "comment"]
            ],
            string: [
                [/[^\\"']+/, "string"],
                [/@escapes/, "string.escape"],
                [/\\./, "string.escape.invalid"],
                [
                    /["']/,
                    {
                        cases: {
                            "$#==$S2": { token: "string", next: "@pop" },
                            "@default": "string"
                        }
                    }
                ]
            ],
            bracketedString: [
                [/[^\]]+/, "string"],
                [
                    /\](=*)\]/,
                    {
                        cases: {
                            "$1==$S2": { token: "string", next: "@pop" },
                            "@default": "string"
                        }
                    }
                ],
                [/./, "string"]
            ]
        }
    };

    return { conf: conf, language: language };
});
