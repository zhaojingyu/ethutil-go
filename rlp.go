package ethutil

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"math/big"
	"reflect"
)

///////////////////////////////////////
type EthEncoder interface {
	EncodeData(rlpData interface{}) []byte
}
type EthDecoder interface {
	Get(idx int) *RlpDataAttribute
}

//////////////////////////////////////

type RlpEncoder struct {
	rlpData []byte
}

func NewRlpEncoder() *RlpEncoder {
	encoder := &RlpEncoder{}

	return encoder
}
func (coder *RlpEncoder) EncodeData(rlpData interface{}) []byte {
	return Encode(rlpData)
}

// Data attributes are returned by the rlp decoder. The data attributes represents
// one item within the rlp data structure. It's responsible for all the casting
// It always returns something valid
type RlpDataAttribute struct {
	dataAttrib interface{}
	kind       reflect.Value
}

func Conv(attrib interface{}) *RlpDataAttribute {
	return &RlpDataAttribute{dataAttrib: attrib, kind: reflect.ValueOf(attrib)}
}

func NewRlpDataAttribute(attrib interface{}) *RlpDataAttribute {
	return &RlpDataAttribute{dataAttrib: attrib}
}

func (attr *RlpDataAttribute) Type() reflect.Kind {
	return reflect.TypeOf(attr.dataAttrib).Kind()
}

func (attr *RlpDataAttribute) IsNil() bool {
	return attr.dataAttrib == nil
}

func (attr *RlpDataAttribute) Length() int {
	//return attr.kind.Len()
	if data, ok := attr.dataAttrib.([]interface{}); ok {
		return len(data)
	}

	return 0
}

func (attr *RlpDataAttribute) AsRaw() interface{} {
	return attr.dataAttrib
}

func (attr *RlpDataAttribute) AsUint() uint64 {
	if value, ok := attr.dataAttrib.(uint8); ok {
		return uint64(value)
	} else if value, ok := attr.dataAttrib.(uint16); ok {
		return uint64(value)
	} else if value, ok := attr.dataAttrib.(uint32); ok {
		return uint64(value)
	} else if value, ok := attr.dataAttrib.(uint64); ok {
		return value
	}

	return 0
}

func (attr *RlpDataAttribute) AsByte() byte {
	if value, ok := attr.dataAttrib.(byte); ok {
		return value
	}

	return 0x0
}

func (attr *RlpDataAttribute) AsBigInt() *big.Int {
	if a, ok := attr.dataAttrib.([]byte); ok {
		b := new(big.Int)
		b.SetString(string(a), 0)
		return b
	}

	return big.NewInt(0)
}

func (attr *RlpDataAttribute) AsString() string {
	if a, ok := attr.dataAttrib.([]byte); ok {
		return string(a)
	} else if a, ok := attr.dataAttrib.(string); ok {
		return a
	} else {
		//panic(fmt.Sprintf("not string %T: %v", attr.dataAttrib, attr.dataAttrib))
	}

	return ""
}

func (attr *RlpDataAttribute) AsBytes() []byte {
	if a, ok := attr.dataAttrib.([]byte); ok {
		return a
	}

	return make([]byte, 0)
}

func (attr *RlpDataAttribute) AsSlice() []interface{} {
	if d, ok := attr.dataAttrib.([]interface{}); ok {
		return d
	}

	return []interface{}{}
}

// Threat the attribute as a slice
func (attr *RlpDataAttribute) Get(idx int) *RlpDataAttribute {
	if d, ok := attr.dataAttrib.([]interface{}); ok {
		// Guard for oob
		if len(d) < idx {
			return NewRlpDataAttribute(nil)
		}

		return NewRlpDataAttribute(d[idx])
	}

	// If this wasn't a slice you probably shouldn't be using this function
	return NewRlpDataAttribute(nil)
}

type RlpDecoder struct {
	rlpData interface{}
}

func NewRlpDecoder(rlpData []byte) *RlpDataAttribute {
	//decoder := &RlpDecoder{}
	// Decode the data

	if len(rlpData) != 0 {
		data, _ := Decode(rlpData, 0)
		//decoder.rlpData = data
		return NewRlpDataAttribute(data)
	}

	return NewRlpDataAttribute(nil)
}

func (dec *RlpDecoder) Get(idx int) *RlpDataAttribute {
	return NewRlpDataAttribute(dec.rlpData).Get(idx)
}

/// Raw methods
func BinaryLength(n uint64) uint64 {
	if n == 0 {
		return 0
	}

	return 1 + BinaryLength(n/256)
}

func ToBinarySlice(n uint64, length uint64) []uint64 {
	if length == 0 {
		length = BinaryLength(n)
	}

	if n == 0 {
		return []uint64{}
	}

	slice := ToBinarySlice(n/256, 0)
	slice = append(slice, n%256)

	return slice
}

func ToBin(n uint64, length uint64) string {
	var buf bytes.Buffer
	for _, val := range ToBinarySlice(n, length) {
		buf.WriteByte(byte(val))
	}

	return buf.String()
}

func FromBin(data []byte) uint64 {
	if len(data) == 0 {
		return 0
	}

	return FromBin(data[:len(data)-1])*256 + uint64(data[len(data)-1])
}

func Decode(data []byte, pos uint64) (interface{}, uint64) {
	if pos > uint64(len(data)-1) {
		log.Println(data)
		log.Panicf("index out of range %d for data %q, l = %d", pos, data, len(data))
	}

	char := int(data[pos])
	slice := make([]interface{}, 0)
	switch {
	case char < 24:
		return data[pos], pos + 1

	case char < 56:
		b := uint64(data[pos]) - 23
		return FromBin(data[pos+1 : pos+1+b]), pos + 1 + b

	case char < 64:
		b := uint64(data[pos]) - 55
		b2 := uint64(FromBin(data[pos+1 : pos+1+b]))
		return FromBin(data[pos+1+b : pos+1+b+b2]), pos + 1 + b + b2

	case char < 120:
		b := uint64(data[pos]) - 64
		return data[pos+1 : pos+1+b], pos + 1 + b

	case char < 128:
		b := uint64(data[pos]) - 119
		b2 := uint64(FromBin(data[pos+1 : pos+1+b]))
		return data[pos+1+b : pos+1+b+b2], pos + 1 + b + b2

	case char < 184:
		b := uint64(data[pos]) - 128
		pos++
		for i := uint64(0); i < b; i++ {
			var obj interface{}

			obj, pos = Decode(data, pos)
			slice = append(slice, obj)
		}
		return slice, pos

	case char < 192:
		b := uint64(data[pos]) - 183
		//b2 := int(FromBin(data[pos+1 : pos+1+b])) (ref implementation has an unused variable)
		pos = pos + 1 + b
		for i := uint64(0); i < b; i++ {
			var obj interface{}

			obj, pos = Decode(data, pos)
			slice = append(slice, obj)
		}
		return slice, pos

	default:
		panic(fmt.Sprintf("byte not supported: %q", char))
	}

	return slice, 0
}

func Encode(object interface{}) []byte {
	var buff bytes.Buffer

	if object != nil {
		switch t := object.(type) {
		case int:
			buff.Write(Encode(uint32(t)))
		case uint32, uint64:
			var num uint64
			if _num, ok := t.(uint64); ok {
				num = _num
			} else if _num, ok := t.(uint32); ok {
				num = uint64(_num)
			}

			if num >= 0 && num < 24 {
				buff.WriteString(string(num))
			} else if num <= uint64(math.Pow(2, 256)) {
				b := ToBin(num, 0)
				buff.WriteString(string(len(b)+23) + b)
			} else {
				b := ToBin(num, 0)
				b2 := ToBin(uint64(len(b)), 0)
				buff.WriteString(string(len(b2)+55) + b2 + b)
			}

		case *big.Int:
			buff.Write(Encode(t.String()))

		case string:
			if len(t) < 56 {
				buff.WriteString(string(len(t)+64) + t)
			} else {
				b2 := ToBin(uint64(len(t)), 0)
				buff.WriteString(string(len(b2)+119) + b2 + t)
			}

		case byte:
			buff.Write(Encode(uint32(t)))
		case []byte:
			// Cast the byte slice to a string
			buff.Write(Encode(string(t)))

		case []interface{}, []string:
			// Inline function for writing the slice header
			WriteSliceHeader := func(length int) {
				if length < 56 {
					buff.WriteByte(byte(length + 128))
				} else {
					b2 := ToBin(uint64(length), 0)
					buff.WriteByte(byte(len(b2) + 183))
					buff.WriteString(b2)
				}
			}

			// FIXME How can I do this "better"?
			if interSlice, ok := t.([]interface{}); ok {
				WriteSliceHeader(len(interSlice))
				for _, val := range interSlice {
					buff.Write(Encode(val))
				}
			} else if stringSlice, ok := t.([]string); ok {
				WriteSliceHeader(len(stringSlice))
				for _, val := range stringSlice {
					buff.Write(Encode(val))
				}
			}
		}
	} else {
		// Write an empty string if the object was nil
		buff.Write(Encode(""))
	}

	return buff.Bytes()
}
