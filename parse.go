package monster
import "text/scanner"
import "os"
import "strings"
import "fmt"
import "strconv"
import "math/rand"

type Node interface{}
type Context map[string]interface{}

// Types of Terminal as indicated by `name` field,
//  BnfRange, BnfBag
type Terminal struct {
    name string
    value string
    generator func(context Context) string
}
type NonTerminal struct {
    name string
    value string
    context Context
    lrmax int
    generator func(context Context) string
    any [][]Node
}
type ParseOpts struct {
    Prodfile string
    Rnd *rand.Rand
    S *scanner.Scanner
    Nonterminals map[string]NonTerminal
}

// Global variables
var Literals = make( map[string]func(string)Terminal )  // Literal handlers
var Terminals = make( map[string]func()Terminal )       // Terminal handlers
var Bnfs = make( map[string]func(ParseOpts)Terminal )   // Built-in functions

func Token(s *scanner.Scanner) string {
    s.Scan()
    return s.TokenText()
}

func Tokent(s *scanner.Scanner) (string, string) {
    return scanner.TokenString(s.Scan()), s.TokenText()
}

func Parse(popts *ParseOpts) string {
    var s scanner.Scanner
    popts.Nonterminals = make( map[string]NonTerminal )
    file, err := os.Open(popts.Prodfile)
    if err != nil {
        fmt.Printf("Error with file")
    }
    popts.S = &s
    s.Init(file)
    ntname, lrmax := parseNT( popts )
    start := ntname
    for ntname != "" {
        if ntname == "Context" { break }
        nt := parseRules(popts, ntname)
        nt.lrmax, popts.Nonterminals[ntname] = lrmax, nt
        ntname, lrmax = parseNT(popts)
    }

    // Parse context information
    if ntname == "Context" {
        if Token( popts.S ) != "." {
            fmt.Printf("Context statement must end with a Dot(.)\n")
            os.Exit(1)
        }
        parseContext(popts)
    }
    buildAst(popts)
    return start
}

// parse nonterminal tokens
func parseNT( popts *ParseOpts ) (string, int) {
    var lrmax int = -1
    _, ntname := Tokent(popts.S)
    ch := popts.S.Peek()
    if ch == '!' || ch == '#' {
        _ = Token(popts.S)
        tok3 := Token(popts.S)
        lrmax, _ = strconv.Atoi(tok3)
    }
    return ntname, lrmax
}

// Parse alternate rules for nonterminal `ntname`
func parseRules( popts *ParseOpts, ntname string ) NonTerminal {
    var any [100][]Node
    next, i := true, 0
    for ;next; i++ {
        tok := Token(popts.S)
        if tok != ":" && tok != "|" { 
            fmt.Printf( "A rule should begin with : or |\n" )
            os.Exit(1)
        }
        any[i], next = parseNodes(popts)
    }
    nt := NonTerminal{ name: ntname, any: any[:i], context: make(Context) }
    return nt
}

// Parse rule
func parseNodes( popts *ParseOpts ) ([]Node, bool) {
    var all [100]Node
    var i int
    toktype, tok := Tokent(popts.S)
    for ; (tok != ".") && (tok != ";") && (toktype != "EOF") ; i++ {
        litfn := Literals[toktype]
        termfn := Terminals[tok]
        bnffn := Bnfs[tok]
        if litfn != nil {
            all[i] = litfn(tok)
        } else if termfn != nil {
            all[i] = termfn()
        } else if bnffn != nil {
            all[i] = bnffn( *popts )
        } else if tok == "$" {
            tok = Token(popts.S)
            all[i] = Literals["Reference"]( tok )
        } else {
            all[i] = tok
        }
        toktype, tok = Tokent(popts.S)
    }
    return all[:i], tok == ";"
}


func parseContext( popts *ParseOpts ) {
    var value interface{}
    for {
        toktype1, nonterm := Tokent(popts.S)
        _, tok2 := Tokent(popts.S)
        if toktype1 != "Ident" && tok2 != ":" { break; }
        for {
            toktype1, tok1 := Tokent(popts.S)
            _, tok2 := Tokent(popts.S)
            toktype3, tok3 := Tokent(popts.S)
            _, tok4 := Tokent(popts.S)
            if toktype1 != "Ident" && tok2 != "=" {
                fmt.Printf( "Invalid context for non-terminal %v\n", nonterm)
                os.Exit(1)
            }
            switch toktype3 {
            case "String" : value = tok3
            case "Char" : value, _ = strconv.ParseInt(tok3, 10, 8)
            case "Int" : value, _ = strconv.ParseInt(tok3, 10, 64)
            case "Float" : value, _ = strconv.ParseFloat(tok3, 64)
            default :
                fmt.Printf( "Invalid context\n")
                os.Exit(1)
            }
            popts.Nonterminals[nonterm].context[tok1] = value
            if tok4 == "." { break }
        }
    }
}

func PrintAst( nonterminals map[string]NonTerminal ) {
    for ntname, nt := range nonterminals {
        fmt.Printf( "%v \n", ntname )
        for _, rule := range nt.any {
            printRule(rule)
        }
    }
}

func printRule( ns []Node ) {
    var outstr = make( []string, len(ns) )
    fmt.Printf("    : ")
    for i, n := range ns {
        outstr[i] = printNode(n)
    }
    fmt.Printf("%v\n", strings.Join(outstr, " . "))
    fmt.Printf("\n")
}

func printNode( n Node ) string {
    valt, ok := n.(Terminal)
    if ok {
        return fmt.Sprintf("%v(%v)", valt.name, valt.value)
    }
    valnt, ok := n.(NonTerminal)
    if ok {
        return fmt.Sprintf("%v", valnt.name)
    }
    fmt.Printf("Unknown Node %v", n)
    os.Exit(1)
    return ""
}

func Generate( context Context, popts ParseOpts, nt NonTerminal ) string {
    var valstr string
    rule := nt.any[ popts.Rnd.Intn(len(nt.any)) ] // Randomly pick a rule
    terms := make( []string, len(rule) )
    names := make( []string, len(rule) )
    for k, v := range popts.Nonterminals[nt.name].context {
        context[k] = v
    }
    for _, node := range rule  {
        nd, ok := node.(Terminal)
        if ok { // Node is terminal
            // fmt.Printf("-- terminal `%v`\n", nd.name)
            valstr = nd.generator(context)
        } else { // Node is nonterminal
            nd, _ := node.(NonTerminal)
            valstr = Generate(context, popts, nd)
            context[nd.name] = valstr
            names = append(names, nd.name)
            // fmt.Printf("-- nonterminal `%v`\n", nd.name)
        }
        terms = append(terms, valstr)
    }
    for _, name := range names { delete(context, name) }
    for k := range popts.Nonterminals[nt.name].context { delete(context, k) }
    return strings.Join( terms, "" )
}

// Build AST from map of `nonterminals`.
func buildAst( popts *ParseOpts ) {
    for _, nt := range popts.Nonterminals {
        for i, rule := range nt.any {
            for j, node := range rule {
                ref, ok := node.(string)
                if ok {
                    if popts.Nonterminals[ref].name == "" {
                        fmt.Printf("Unknown reference %v \n", ref)
                        os.Exit(1)
                    }
                    nt.any[i][j] = popts.Nonterminals[ref]
                }
            }
        }
    }
}

func contains( key string, arrs []string ) bool {
    for _, v := range arrs {
        if key == v { return true }
    }
    return false
}

func nt( m map[string]NonTerminal ) []string {
    var keys = make([]string, len(m))
    i := 0
    for key := range m {
        keys[i] = key
        i++
    }
    return keys
}
