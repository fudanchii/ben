package ben_test

import (
	"bufio"
	"bytes"
	"io"
	"testing"

	"github.com/fudanchii/ben"
	//nolint: revive // BDD sucks
	. "github.com/onsi/gomega"
)

func FuzzDecode(f *testing.F) {
	f.Add("i23e")
	f.Add("0")
	f.Add("d0:e")
	f.Add("d0:0:e")
	f.Add("d1:a1:be")
	f.Add("li1234:abcde")

	//nolint: lll
	f.Add("d7:comment19:now you know my abc10:created by3:abc13:creation datei1759841977e4:infod5:filesld6:lengthi6e4:pathl5:a.txteed6:lengthi6e4:pathl5:b.txteed6:lengthi6e4:pathl5:c.txteee4:name3:abc12:piece lengthi16384e6:pieces20:\x0a\xcbI)1\xec\xcd\x8b\x17\xef\x90\xec\xa6o\x94bqP<\x077:privatei1eee")

	f.Fuzz(func(t *testing.T, input string) {
		g := NewWithT(t)
		l, err := ben.InferredTypeDecode(bufio.NewReader(bytes.NewBufferString(input)))

		if err != nil {
			g.Expect(err).To(SatisfyAny(
				MatchError(io.EOF),
				MatchError(ben.ErrUnknownTypeMarker),
				MatchError(ben.ErrInvalidStringLength),
				MatchError(ben.ErrInputIsNotInteger),
				MatchError(ben.ErrKeyWithoutValue),
				MatchError(ben.ErrEndItemSequence),
			))
		} else {
			g.Expect(l).NotTo(BeNil())
		}
	})
}
