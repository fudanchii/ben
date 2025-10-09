package ben_test

//nolint:gci // huh?
import (
	"bufio"
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/fudanchii/ben"
	"github.com/fudanchii/infr"

	//nolint:revive // due to ginkgo convention
	. "github.com/onsi/ginkgo/v2"

	//nolint:revive // due to ginkgo convention
	. "github.com/onsi/gomega"
)

func TestBen(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ben Suite")
}

var _ = Describe("Bencode", func() {
	Describe("Integer Type", func() {
		It("can encode integer", func() {
			Expect(ben.Int(12).Encode()).To(Equal([]byte("i12e")))
		})

		Context("decoding integer", func() {
			var (
				source *bytes.Buffer
				bInt   ben.Element
				err    error
			)

			BeforeEach(func() {
				source = bytes.NewBufferString("i71183928e")
				bInt, err = ben.InferredTypeDecode(bufio.NewReader(source))
			})

			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should have Integer type", func() {
				Expect(bInt.Type()).To(Equal(ben.IntType))
			})

			It("should returns 71183928", func() {
				Expect(bInt).To(Equal(ben.Int(71183928)))
			})
		})

		It("returns EOF when input ends without `e` delimiter", func() {
			source := bytes.NewBufferString("i1231")
			_, err := ben.InferredTypeDecode(bufio.NewReader(source))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("EOF"))
		})

		It("returns InvalidInputError on invalid input", func() {
			source := bytes.NewBufferString("i9809d3:abc")
			_, err := ben.InferredTypeDecode(bufio.NewReader(source))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Invalid Input Error: 100"))
		})
	})

	Describe("String Type", func() {
		It("can encode string", func() {
			Expect(ben.Str("Hello World!").Encode()).To(Equal([]byte("12:Hello World!")))
		})

		Context("decoding string", func() {
			var (
				source *bytes.Buffer
				bStr   ben.String
				err    error
			)

			BeforeEach(func() {
				source = bytes.NewBufferString("9:Mi Amigos")
				bStr, err = ben.Decode[ben.String](bufio.NewReader(source))
			})

			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should have String type", func() {
				Expect(bStr.Type()).To(Equal(ben.StringType))
			})

			It("should returns Mi Amigos", func() {
				Expect(bStr.Val).To(Equal("Mi Amigos"))
			})
		})

		Context("given extraneous input", func() {
			var (
				source *bytes.Buffer
				bStr   ben.String
				err    error
			)

			BeforeEach(func() {
				source = bytes.NewBufferString("3:abcdef")
				bStr, err = ben.Decode[ben.String](bufio.NewReader(source))
			})

			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should returns abc", func() {
				Expect(bStr.Val).To(Equal("abc"))
			})
		})

		It("returns EOF when input is less than the indicated length", func() {
			source := bytes.NewBufferString("4:abc")
			_, err := ben.Decode[ben.String](bufio.NewReader(source))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("EOF"))
		})
	})

	Describe("List Type", func() {
		It("can encode list", func() {
			list := ben.Lst([]ben.Element{
				ben.Int(42),
				ben.Str("Hello"),
			})
			Expect(list.Encode()).To(Equal([]byte("li42e5:Helloe")))
		})

		Context("decoding list", func() {
			var (
				source  *bytes.Buffer
				bList   ben.Element
				fail    error
				lElm    ben.List
				castErr error
			)

			BeforeEach(func() {
				source = bytes.NewBufferString("li42ei100ei0ei12e4:namee")
				bList, fail = ben.InferredTypeDecode(bufio.NewReader(source))
				lElm, castErr = infr.TryInto[ben.List](bList)
			})

			It("should not cause error", func() {
				Expect(fail).NotTo(HaveOccurred())
			})

			It("should be able to cast without error", func() {
				Expect(castErr).NotTo(HaveOccurred())
			})

			It("should have List type", func() {
				Expect(bList.Type()).To(Equal(ben.ListType))
			})

			It("should have 5 elements", func() {
				Expect(lElm.Val).To(HaveLen(5))
			})

			It("should returns 42 for the first element", func() {
				Expect(lElm.Val[0]).To(Equal(ben.Int(42)))
			})

			It("should returns name for the last element", func() {
				Expect(lElm.Val[4]).To(Equal(ben.Str("name")))
			})
		})

		It("returns EOF when input is ended too early", func() {
			source := bytes.NewBufferString("li234e")
			_, err := ben.Decode[ben.List](bufio.NewReader(source))
			Expect(err.Error()).To(Equal("EOF"))
		})

		Context("decoding partially corrupted list", func() {
			var (
				bList ben.Element
				err   error
			)

			BeforeEach(func() {
				source := bytes.NewBufferString("li234e4:abcdi24")
				bList, err = ben.InferredTypeDecode(bufio.NewReader(source))
			})

			It("should error", func() {
				Expect(err).To(HaveOccurred())
			})

			It("returns non nil value", func() {
				Expect(bList).NotTo(BeNil())
			})

			Context("convert to ben.List", func() {
				var (
					lst ben.List
				)

				BeforeEach(func() {
					lst, err = infr.TryInto[ben.List](bList)
				})

				It("should not error", func() {
					Expect(err).NotTo(HaveOccurred())
				})

				It("should have the first two element", func() {
					Expect(lst.Val).To(HaveLen(2))
				})

				It("should returns 234 for the first element", func() {
					Expect(lst.Val[0]).To(Equal(ben.Int(234)))
				})

				It("should returns abcd for the second element", func() {
					Expect(lst.Val[1]).To(Equal(ben.Str("abcd")))
				})
			})
		})
	})

	Describe("Dictionary type", func() {
		It("can encode Dictionary", func() {
			dict := ben.Dct(map[string]ben.Element{
				"answer":   ben.Int(42),
				"question": ben.Str("to be determined"),
			})

			sampleBenStr1 := []byte("d6:answeri42e8:question16:to be determinede")
			sampleBenStr2 := []byte("d8:question16:to be determined6:answeri42ee")

			benStr := dict.Encode()

			if string(benStr[:2]) == "d6" {
				Expect(benStr).To(Equal(sampleBenStr1))
			}

			if string(benStr[:2]) == "d8" {
				Expect(benStr).To(Equal(sampleBenStr2))
			}
		})

		Context("decoding dictionary", func() {
			var (
				bDict ben.Element
				err   error
			)

			BeforeEach(func() {
				source := bytes.NewBufferString("d6:answeri42e8:question16:to be determinede")
				bDict, err = ben.InferredTypeDecode(bufio.NewReader(source))
			})

			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			Context("casting to ben.Dictionary", func() {
				var (
					d ben.Dictionary
				)

				BeforeEach(func() {
					d, err = infr.TryInto[ben.Dictionary](bDict)
				})

				It("should not error", func() {
					Expect(err).NotTo(HaveOccurred())
				})

				It("should have dictionary type", func() {
					Expect(bDict.Type()).To(Equal(ben.DictType))
				})

				It("should have 2 elements", func() {
					Expect(d.Val).To(HaveLen(2))
				})

				It("should have key: 'answer', with value: '42'", func() {
					Expect(d.Val["answer"]).To(Equal(ben.Int(42)))
				})

				It("should have key: 'question', with value: 'to be determined'", func() {
					Expect(d.Val["question"]).To(Equal(ben.Str("to be determined")))
				})
			})
		})
	})

	Describe("Read torrent file", func() {
		Context("decoding torrent file", func() {
			var (
				content *os.File
				err     error
			)

			Context("cast to ben.Dictionary", func() {
				var (
					torrentDict ben.Dictionary
					torrent     ben.Torrent
				)

				BeforeEach(func() {
					content, err = os.OpenFile("testdata/NetBSD-10.0-amd64.iso.torrent", os.O_RDONLY, 0)
					Expect(err).NotTo(HaveOccurred())

					torrentDict, err = ben.Decode[ben.Dictionary](bufio.NewReader(content))
					Expect(err).NotTo(HaveOccurred())

					torrent, err = infr.TryFrom[ben.Dictionary, ben.Torrent](torrentDict).TryInto()
					Expect(err).NotTo(HaveOccurred())
				})

				AfterEach(func() {
					_ = content.Close()
				})

				It("should have the correct 'announce' value", func() {
					Expect(torrent.Announce).To(Equal("http://tracker.NetBSD.org:6969/announce"))
				})

				It("should have the correct 'created by' value", func() {
					Expect(torrent.CreatedBy).NotTo(BeNil())
					Expect(*torrent.CreatedBy).To(Equal("Transmission/4.0.3 (6b0e49bbb2)"))
				})

				It("should have the correct 'encoding' value", func() {
					Expect(torrent.Encoding).NotTo(BeNil())
					Expect(*torrent.Encoding).To(Equal("UTF-8"))
				})

				It("should  have the correct 'creation date' value", func() {
					Expect(torrent.CreationDate).NotTo(BeNil())

					ts, parseErr := time.Parse(time.RFC3339, "2024-03-30T17:17:24+09:00")
					Expect(parseErr).NotTo(HaveOccurred())

					Expect(*torrent.CreationDate).To(Equal(ts))
				})

				It("should have the correct 'info.name' value", func() {
					Expect(torrent.Info.Name).To(Equal("NetBSD-10.0-amd64.iso"))
				})

				It("should have the correct 'info.length' value", func() {
					Expect(torrent.Info.Length).To(Equal(int64(652652544)))
				})

				It("should have the correct 'info.piece_length' value", func() {
					Expect(torrent.Info.PieceLength).To(Equal(int64(524288)))
				})

				It("should have the correct 'info.pieces' value", func() {
					Expect(torrent.Info.Pieces).To(HaveLen(1245))
					Expect(torrent.Info.Pieces[0]).
						To(Equal(ben.SHA1("\x9a\xe7\x47\x53\x58\x35\x0c\x00\x25\x86\xfe\x2c\x48\x4c\x6c\x62\x66\x10\xb2\x9d")))
				})
			})
		})
	})

	Describe("Read torrent file", func() {
		Context("decoding torrent file", func() {
			var (
				content *os.File
				err     error
			)

			Context("cast to ben.Dictionary", func() {
				var (
					torrentDict ben.Dictionary
					torrent     ben.Torrent
				)

				BeforeEach(func() {
					content, err = os.OpenFile("testdata/abc.torrent", os.O_RDONLY, 0)
					Expect(err).NotTo(HaveOccurred())

					torrentDict, err = ben.Decode[ben.Dictionary](bufio.NewReader(content))
					Expect(err).NotTo(HaveOccurred())

					torrent, err = infr.TryFrom[ben.Dictionary, ben.Torrent](torrentDict).TryInto()
					Expect(err).NotTo(HaveOccurred())
					Expect(torrent).NotTo(BeNil())
				})

				AfterEach(func() { _ = content.Close() })

				It("should have the correct 'announce' value", func() {
					Expect(torrent.Announce).To(Equal(""))
				})

				It("should have the correct 'created by' value", func() {
					Expect(torrent.CreatedBy).NotTo(BeNil())
					Expect(*torrent.CreatedBy).To(Equal("kimbatt.github.io/torrent-creator"))
				})

				It("should have the correct 'encoding' value", func() {
					Expect(torrent.Encoding).To(BeNil())
				})

				It("should  have the correct 'creation date' value", func() {
					Expect(torrent.CreationDate).NotTo(BeNil())

					ts, parseErr := time.Parse(time.RFC3339, "2025-10-07T21:59:37+09:00")
					Expect(parseErr).NotTo(HaveOccurred())

					Expect(*torrent.CreationDate).To(Equal(ts))
				})

				It("should have the correct 'info.name' value", func() {
					Expect(torrent.Info.Name).To(Equal("abc"))
				})

				It("should have the correct 'info.length' value", func() {
					Expect(torrent.Info.Length).To(Equal(int64(0)))
				})

				It("should have the correct 'info.piece_length' value", func() {
					Expect(torrent.Info.PieceLength).To(Equal(int64(16384)))
				})

				It("should have the correct 'info.pieces' value", func() {
					Expect(torrent.Info.Pieces).To(HaveLen(1))
					Expect(torrent.Info.Pieces[0]).
						To(Equal(ben.SHA1(
							[]byte{10, 203, 73, 41, 49, 236, 205, 139, 23, 239, 144, 236, 166, 111, 148, 98, 113, 80, 60, 7},
						)))
				})

				It("should have the correct 'info.files' value", func() {
					Expect(torrent.Info.Files).To(HaveLen(3))

					expectedFiles := []struct {
						length int64
						path   []string
					}{
						{
							length: 6,
							path:   []string{"a.txt"},
						},
						{
							length: 6,
							path:   []string{"b.txt"},
						},
						{
							length: 6,
							path:   []string{"c.txt"},
						},
					}

					for i, file := range torrent.Info.Files {
						Expect(file.Length).To(Equal(expectedFiles[i].length))
						Expect(file.Path).To(Equal(expectedFiles[i].path))
					}
				})
			})
		})
	})
})
