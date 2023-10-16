package ben_test

import (
	"bufio"
	"bytes"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/fudanchii/ben"
	"github.com/fudanchii/infr"
)

func TestBen(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ben Suite")
}

var _ = Describe("Bencode", func() {
	Describe("Integer Type", func() {
		It("can encode integer", func() {
			Expect(NewInteger(12).Encode()).To(Equal([]byte("i12e")))
		})

		Context("decoding integer", func() {
			source := bytes.NewBufferString("i71183928e")
			bInt, err := InferredTypeDecode(bufio.NewReader(source))
			Expect(err).NotTo(HaveOccurred())

			It("should have Integer type", func() {
				Expect(bInt.Type()).To(Equal(IntType))
			})

			It("should returns 71183928", func() {
				Expect(infr.TryInto[Integer](bInt)).To(Equal(Integer(71183928)))
			})
		})

		It("returns EOF when input ends without `e` delimiter", func() {
			source := bytes.NewBufferString("i1231")
			_, err := InferredTypeDecode(bufio.NewReader(source))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("EOF"))
		})

		It("returns InvalidInputError on invalid input", func() {
			source := bytes.NewBufferString("i9809d3:abc")
			_, err := InferredTypeDecode(bufio.NewReader(source))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Invalid Input Error: 100"))
		})
	})

	Describe("String Type", func() {
		It("can encode string", func() {
			Expect(String("Hello World!").Encode()).To(Equal([]byte("12:Hello World!")))
		})

		Context("decoding string", func() {
			source := bytes.NewBufferString("9:Mi Amigos")
			bStr, err := Decode[String](bufio.NewReader(source))
			Expect(err).NotTo(HaveOccurred())

			It("should have String type", func() {
				Expect(bStr.Type()).To(Equal(StringType))
			})

			It("should returns Mi Amigos", func() {
				Expect(string(bStr)).To(Equal("Mi Amigos"))
			})
		})

		Context("given extraneous input", func() {
			source := bytes.NewBufferString("3:abcdef")
			bStr, err := Decode[String](bufio.NewReader(source))
			Expect(err).NotTo(HaveOccurred())

			It("should returns abc", func() {
				Expect(string(bStr)).To(Equal("abc"))
			})
		})

		It("returns EOF when input is less than the indicated length", func() {
			source := bytes.NewBufferString("4:abc")
			_, err := Decode[String](bufio.NewReader(source))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("EOF"))
		})
	})

	Describe("List Type", func() {
		It("can encode list", func() {
			list := NewList([]Element{
				NewInteger(42),
				NewString("Hello"),
			})
			Expect(list.Encode()).To(Equal([]byte("li42e5:Helloe")))
		})

		Context("decoding list", func() {
			source := bytes.NewBufferString("li42ei100ei0ei12e4:namee")
			bList, err := InferredTypeDecode(bufio.NewReader(source))
			Expect(err).NotTo(HaveOccurred())

			It("should have List type", func() {
				Expect(bList.Type()).To(Equal(ListType))
			})

			lElm, err := infr.TryInto[List](bList)
			It("should have []Element type for its data", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should have 5 elements", func() {
				Expect(len(lElm)).To(Equal(5))
			})

			It("should returns 42 for the first element", func() {
				Expect(infr.TryInto[Integer](lElm[0])).To(Equal(Integer(42)))
			})

			It("should returns name for the last element", func() {
				Expect(infr.TryInto[String](lElm[4])).To(Equal(String("name")))
			})
		})

		It("returns EOF when input is ended too early", func() {
			source := bytes.NewBufferString("li234e")
			_, err := Decode[List](bufio.NewReader(source))
			Expect(err.Error()).To(Equal("EOF"))
		})

		Context("decoding partially corrupted list", func() {
			source := bytes.NewBufferString("li234e4:abcdi24")
			bList, err := InferredTypeDecode(bufio.NewReader(source))
			Expect(err).To(HaveOccurred())

			It("returns non nil value", func() {
				Expect(bList).NotTo(BeNil())
			})

			lst, err := infr.TryInto[List](bList)
			Expect(err).NotTo(HaveOccurred())

			It("should have the first two element", func() {
				Expect(len(lst)).To(Equal(2))
			})

			It("should returns 234 for the first element", func() {
				Expect(infr.TryInto[Integer](lst[0])).To(Equal(Integer(234)))
			})

			It("should returns abcd for the second element", func() {
				Expect(infr.TryInto[String](lst[1])).To(Equal(String("abcd")))
			})
		})
	})

	Describe("Dictionary type", func() {
		It("can encode Dictionary", func() {
			dict := NewDictionary(map[string]Element{
				"answer":   NewInteger(42),
				"question": NewString("to be determined"),
			})
			Expect(dict.Encode()).To(Equal([]byte("d6:answeri42e8:question16:to be determinede")))
		})

		Context("decoding dictionary", func() {
			source := bytes.NewBufferString("d6:answeri42e8:question16:to be determinede")
			bDict, err := InferredTypeDecode(bufio.NewReader(source))
			Expect(err).NotTo(HaveOccurred())

			It("should have dictionary type", func() {
				Expect(bDict.Type()).To(Equal(DictType))
			})

			It("should have 2 elements", func() {
				d, err := infr.TryInto[Dictionary](bDict)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(d)).To(Equal(2))
			})

			It("should have key: 'answer', with value: '42'", func() {
				d, err := infr.TryInto[Dictionary](bDict)
				Expect(err).NotTo(HaveOccurred())
				Expect(d["answer"]).To(Equal(Integer(42)))
			})

			It("should have key: 'question', with value: 'to be determined'", func() {
				d, err := infr.TryInto[Dictionary](bDict)
				Expect(err).NotTo(HaveOccurred())
				Expect(d["question"]).To(Equal(String("to be determined")))
			})
		})
	})
})
