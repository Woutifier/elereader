package parser

import (
	"github.com/Woutifier/elereader/schema"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
)

type Telegram struct {
	Datetime   time.Time
	Identifier string  `Init @Ident`
	Lines      []*Line `{ @@ }`
	Checksum   string  `End @Ident`
}

type Line struct {
	Section    string      `@Ident`
	Properties []*Property `{"(" @@ ")"}`
}

type Property struct {
	Value string `@Ident`
	Unit  string `["*" @Ident]`
}

var (
	parser *participle.Parser
)

func init() {
	regexLexer, err := lexer.Regexp("(?P<Init>/)|(?P<Ident>[:A-Za-z0-9.\\\\-]+)|(\n)|(?P<Literal>([\\(\\)*]))|(?P<End>!)")
	if err != nil {
		log.Fatalf("Unable to create lexer: %s", err)
	}

	parser, err = participle.Build(&Telegram{}, participle.Lexer(regexLexer))
	if err != nil {
		log.Printf("Failed to lex: %s", err)
	}
}

func ParseTelegram(body string) (*Telegram, error) {
	telegram := &Telegram{}
	err := parser.ParseString(body, telegram)
	if err != nil {
		return nil, err
	}
	telegram.Datetime = time.Now()
	return telegram, nil
}

func (t *Telegram) GetReading() schema.Reading {
	reading := schema.Reading{}

	for _, line := range t.Lines {
		if line.Section == "1-0:1.8.1.255" {
			reading.ElectricityHigh = toFloat32(line.Properties[0].Value)
		} else if line.Section == "1-0:2.8.1.255" {
			reading.ElectricityLow = toFloat32(line.Properties[0].Value)
		} else if strings.HasSuffix(line.Section, "24.2.1.255") {
			reading.Gas = toFloat32(line.Properties[0].Value)
		}
	}

	reading.Datetime = uint32(t.Datetime.Unix())

	return reading
}

func toFloat32(s string) float32 {
	v, err := strconv.ParseFloat(s, 10)
	if err != nil {
		log.Fatalf("Invalid float: %s, caused error: %s", s, err)
	}
	return float32(v)
}
