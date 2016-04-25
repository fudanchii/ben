package ben_test

import (
	"bytes"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/fudanchii/ben"
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
			bInt, err := Decode(source)

			It("should decodes successfully", func() {
				Expect(err).To(BeNil())
			})

			It("should have Integer type", func() {
				Expect(bInt.Type()).To(Equal(IntType))
			})

			It("should returns 71183928", func() {
				Expect(bInt.Val().(int64)).To(Equal(int64(71183928)))
			})
		})

		It("returns EOF when input ends without `e` delimiter", func() {
			source := bytes.NewBufferString("i1231")
			_, err := Decode(source)
			Expect(err.Error()).To(Equal("EOF"))
		})

		It("returns InvalidInputError on invalid input", func() {
			source := bytes.NewBufferString("i9809d3:abc")
			_, err := Decode(source)
			_, ok := err.(InvalidInputError)
			Expect(ok).To(BeTrue())
		})
	})

	Describe("String Type", func() {
		It("can encode string", func() {
			Expect(NewString("Hello World!").Encode()).To(Equal([]byte("12:Hello World!")))
		})

		Context("decoding string", func() {
			source := bytes.NewBufferString("9:Mi Amigos")
			bStr, err := Decode(source)

			It("should decodes successfully", func() {
				Expect(err).To(BeNil())
			})

			It("should have String type", func() {
				Expect(bStr.Type()).To(Equal(StringType))
			})

			It("should returns Mi Amigos", func() {
				Expect(bStr.Val().(string)).To(Equal("Mi Amigos"))
			})
		})

		Context("given extraneous input", func() {
			source := bytes.NewBufferString("3:abcdef")
			bStr, err := Decode(source)

			It("should decodes successfully", func() {
				Expect(err).To(BeNil())
			})

			It("should returns abc", func() {
				Expect(bStr.Val().(string)).To(Equal("abc"))
			})
		})

		It("returns EOF when input is less than the indicated length", func() {
			source := bytes.NewBufferString("4:abc")
			_, err := Decode(source)
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
			bList, err := Decode(source)

			It("should decodes successfully", func() {
				Expect(err).To(BeNil())
			})

			It("should have List type", func() {
				Expect(bList.Type()).To(Equal(ListType))
			})

			lElm, ok := bList.Val().([]Element)
			It("should have []Element type for its data", func() {
				Expect(ok).To(BeTrue())
			})

			It("should have 5 elements", func() {
				Expect(len(lElm)).To(Equal(5))
			})

			It("should returns 42 for the first element", func() {
				Expect(lElm[0].Val().(int64)).To(Equal(int64(42)))
			})

			It("should returns name for the last element", func() {
				Expect(lElm[4].Val().(string)).To(Equal("name"))
			})
		})

		It("returns EOF when input is ended too early", func() {
			source := bytes.NewBufferString("li234e")
			_, err := Decode(source)
			Expect(err.Error()).To(Equal("EOF"))
		})

		Context("decoding partially corrupted list", func() {
			source := bytes.NewBufferString("li234e4:abcdi24")
			bList, err := Decode(source)

			It("still returns error", func() {
				Expect(err.Error()).To(Equal("EOF"))
			})

			It("returns non nil value", func() {
				Expect(bList).NotTo(BeNil())
			})

			if bList.(*List) != nil {
				lmnt, ok := bList.Val().([]Element)

				It("should have []Element type", func() {
					Expect(ok).To(BeTrue())
				})

				It("should have the first two element", func() {
					Expect(len(lmnt)).To(Equal(2))
				})

				It("should returns 234 for the first element", func() {
					Expect(lmnt[0].Val().(int64)).To(Equal(int64(234)))
				})

				It("should returns abcd for the second element", func() {
					Expect(lmnt[1].Val().(string)).To(Equal("abcd"))
				})
			}
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
			bDict, err := Decode(source)

			It("should decodes succesfully", func() {
				Expect(err).To(BeNil())
			})

			It("should have dictionary type", func() {
				Expect(bDict.Type()).To(Equal(DictType))
			})

			dVal, ok := bDict.Val().(map[string]Element)
			It("should successfully cast its value", func() {
				Expect(ok).To(BeTrue())
			})
			It("should have 2 elements", func() {
				Expect(len(dVal)).To(Equal(2))
			})

		})
	})
})
