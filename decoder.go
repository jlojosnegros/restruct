package restruct

import (
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
)

// Unpacker is a type capable of unpacking a binary representation of itself
// into a native representation. The Unpack function is expected to consume
// a number of bytes from the buffer, then return a slice of the remaining
// bytes in the buffer. You may use a pointer receiver even if the type is
// used by value.
type Unpacker interface {
	Unpack(buf []byte, order binary.ByteOrder) ([]byte, error)
}

type decoder struct {
	order      binary.ByteOrder
	buf        []byte
	struc      reflect.Value
	sfields    []field
	bitCounter uint8
}

func (d *decoder) readBits(f field, outputLength uint8) (output []byte) {
	// NOTE: We can calculate outputLength but we ask it as a parameter by now
	// outputLength := uint8(t.Expected.Type().Bits() / 8)

	// NOTE: outputLength is the caller desired size independently of the data size
	// So we can have a 3 bits data into a uint64 variable ...
	fmt.Println("----------readBits--------")
	fmt.Printf("incomingData: ")
	for _, elem := range d.buf {
		fmt.Printf("0x%X ", elem)
	}
	fmt.Println()
	fmt.Println("outputLength:", outputLength)
	output = make([]byte, outputLength)

	if f.BitSize == 0 {
		f.BitSize = uint8(f.Type.Bits())
	}

	// Case A: there is no bitfileds => Use legacy code
	if d.bitCounter == 0 && (f.BitSize%8) == 0 {
		fmt.Println("CASE A: NO bitfields")
		// calculate output
		output = d.buf[0:outputLength]
		fmt.Println("output:", output)
		// update buffer
		d.buf = d.buf[outputLength:]
		fmt.Println("rest:", d.buf)
		// no need to update bitCounter
		return
	}

	maskShift := f.BitSize % 8
	var mask uint8 = (0x01 << maskShift) - 1
	fmt.Printf("maskshift:%d\n", maskShift)
	fmt.Printf("mask:0x%X\n", mask)

	fmt.Printf("BitSize:%d\n", f.BitSize)
	fmt.Printf("BitCounter:%d\n", d.bitCounter)

	// Case B: If we just need one byte
	if d.bitCounter+f.BitSize <= 8 {
		fmt.Println("CASE B: Less than a byte")
		output[len(output)-1] = (d.buf[0] >> (8 - (f.BitSize + d.bitCounter))) & mask
		d.bitCounter = (d.bitCounter + f.BitSize) % 8
		fmt.Println("output:", output)
		fmt.Println("rest:", d.buf)
		fmt.Printf("BitCounter:%d\n", d.bitCounter)
		return
	}
	fmt.Println("CASE C:Complex bitfields")
	// Case C: if we need more than one byte
	// first of all calculate the number of actual bytes we need.
	numBytesToUse := ((d.bitCounter + f.BitSize) / 8)
	if (d.bitCounter+f.BitSize)%8 != 0 {
		numBytesToUse++
	}
	// and check the input buffer
	if uint8(len(d.buf)) < numBytesToUse {
		panic(fmt.Errorf("not enought bytes: needed %d, input has %d",
			numBytesToUse,
			len(d.buf)))
	}

	// Calculate the actual length of the data
	// NOTE: This depends on the number of bits, not the caller desired size
	dataLength := f.BitSize / 8
	if (f.BitSize % 8) != 0 {
		dataLength++
	}

	shift := (f.BitSize - (8 - d.bitCounter)) % 8

	// Copy data
	// NOTE: As there could be a difference between data size and the client
	// desired output size we need two different indexes
	outputIdx := (outputLength - dataLength)
	for idx := uint8(0); idx < dataLength; idx++ {
		output[outputIdx] = (d.buf[idx] << shift) | (d.buf[idx+1] >> (8 - shift))
		outputIdx++
	}
	// mask calculation

	if maskShift != 0 {
		var mask uint8 = (0x01 << maskShift) - 1
		output[0] &= mask
	}
	// update IncomingData (use -1 due to base zero index)
	d.buf = d.buf[numBytesToUse-1:]
	// update BitCounter
	d.bitCounter = (f.BitSize + d.bitCounter) % 8

	fmt.Println("output:", output)
	fmt.Println("rest:", d.buf)
	fmt.Printf("BitCounter:%d\n", d.bitCounter)
	return
}

func (d *decoder) read8(f field) uint8 {

	rawdata := d.readBits(f, 1)
	fmt.Printf("read8: rawData=")
	for _, elem := range rawdata {
		fmt.Printf("0x%X ", elem)
	}
	fmt.Println()

	return uint8(rawdata[0])
	// if d.bitCounter == 0 && (f.BitSize%8) == 0 {
	// 	x := d.buf[0]
	// 	d.buf = d.buf[1:]
	// 	return x
	// }

	//	panic(fmt.Errorf("not implemented"))
}

func (d *decoder) read16(f field) uint16 {
	rawdata := d.readBits(f, 2)
	fmt.Printf("read16: rawData=")
	for _, elem := range rawdata {
		fmt.Printf("0x%X ", elem)
	}
	fmt.Println()

	return d.order.Uint16(rawdata)

	// if d.bitCounter == 0 && f.BitSize == 0 {
	// 	x := d.order.Uint16(d.buf[0:2])
	// 	d.buf = d.buf[2:]
	// 	return x
	// }
	// panic(fmt.Errorf("not implemented"))
}

func (d *decoder) read32(f field) uint32 {
	rawdata := d.readBits(f, 4)
	fmt.Printf("read32: rawData=")
	for _, elem := range rawdata {
		fmt.Printf("0x%X ", elem)
	}
	fmt.Println()

	return d.order.Uint32(rawdata)
	// if d.bitCounter == 0 && f.BitSize == 0 {
	// 	x := d.order.Uint32(d.buf[0:4])
	// 	d.buf = d.buf[4:]
	// 	return x
	// }
	// panic(fmt.Errorf("not implemented"))
}

func (d *decoder) read64(f field) uint64 {

	rawdata := d.readBits(f, 8)
	fmt.Printf("read64: rawData=")
	for _, elem := range rawdata {
		fmt.Printf("0x%X ", elem)
	}
	fmt.Println()

	return d.order.Uint64(rawdata)

	// if d.bitCounter == 0 && f.BitSize == 0 {
	// 	x := d.order.Uint64(d.buf[0:8])
	// 	d.buf = d.buf[8:]
	// 	return x
	// }
	// panic(fmt.Errorf("not implemented"))
}

func (d *decoder) readS8(f field) int8 { return int8(d.read8(f)) }

func (d *decoder) readS16(f field) int16 { return int16(d.read16(f)) }

func (d *decoder) readS32(f field) int32 { return int32(d.read32(f)) }

func (d *decoder) readS64(f field) int64 { return int64(d.read64(f)) }

func (d *decoder) readn(count int) []byte {
	x := d.buf[0:count]
	d.buf = d.buf[count:]
	return x
}

func (d *decoder) skipn(count int) {
	d.buf = d.buf[count:]
}

func (d *decoder) skip(f field, v reflect.Value) {
	d.skipn(f.SizeOf(v))
}

func (d *decoder) unpacker(v reflect.Value) (Unpacker, bool) {
	if s, ok := v.Interface().(Unpacker); ok {
		return s, true
	}

	if !v.CanAddr() {
		return nil, false
	}

	if s, ok := v.Addr().Interface().(Unpacker); ok {
		return s, true
	}

	return nil, false
}

func (d *decoder) read(f field, v reflect.Value) {
	if f.Name != "_" {
		if s, ok := d.unpacker(v); ok {
			var err error
			d.buf, err = s.Unpack(d.buf, d.order)
			if err != nil {
				panic(err)
			}
			return
		}
	} else {
		d.skipn(f.SizeOf(v))
		return
	}

	struc := d.struc
	sfields := d.sfields
	order := d.order

	if f.Order != nil {
		d.order = f.Order
		defer func() { d.order = order }()
	}

	if f.Skip != 0 {
		d.skipn(f.Skip)
	}

	switch f.Type.Kind() {
	case reflect.Array:
		l := f.Type.Len()

		// If the underlying value is a slice, initialize it.
		if f.DefType.Kind() == reflect.Slice {
			v.Set(reflect.MakeSlice(reflect.SliceOf(f.Type.Elem()), l, l))
		}

		switch f.DefType.Kind() {
		case reflect.String:
			v.SetString(string(d.readn(f.SizeOf(v))))
		case reflect.Slice, reflect.Array:
			ef := f.Elem()
			for i := 0; i < l; i++ {
				d.read(ef, v.Index(i))
			}
		default:
			panic(fmt.Errorf("invalid array cast type: %s", f.DefType.String()))
		}

	case reflect.Struct:
		d.struc = v
		d.sfields = cachedFieldsFromStruct(f.Type)
		l := len(d.sfields)
		for i := 0; i < l; i++ {
			f := d.sfields[i]
			v := v.Field(f.Index)
			if v.CanSet() {
				d.read(f, v)
			} else {
				d.skip(f, v)
			}
		}
		d.sfields = sfields
		d.struc = struc

	case reflect.Slice, reflect.String:
		switch f.DefType.Kind() {
		case reflect.String:
			l := v.Len()
			v.SetString(string(d.readn(l)))
		case reflect.Slice, reflect.Array:
			switch f.DefType.Elem().Kind() {
			case reflect.Uint8:
				v.SetBytes(d.readn(f.SizeOf(v)))
			default:
				l := v.Len()
				ef := f.Elem()
				for i := 0; i < l; i++ {
					d.read(ef, v.Index(i))
				}
			}
		default:
			panic(fmt.Errorf("invalid array cast type: %s", f.DefType.String()))
		}

	case reflect.Int8:
		v.SetInt(int64(d.readS8(f)))
	case reflect.Int16:
		v.SetInt(int64(d.readS16(f)))
	case reflect.Int32:
		v.SetInt(int64(d.readS32(f)))
	case reflect.Int64:
		v.SetInt(d.readS64(f))

	case reflect.Uint8:
		v.SetUint(uint64(d.read8(f)))
	case reflect.Uint16:
		v.SetUint(uint64(d.read16(f)))
	case reflect.Uint32:
		v.SetUint(uint64(d.read32(f)))
	case reflect.Uint64:
		v.SetUint(d.read64(f))

	case reflect.Float32:
		v.SetFloat(float64(math.Float32frombits(d.read32(f))))
	case reflect.Float64:
		v.SetFloat(math.Float64frombits(d.read64(f)))

	case reflect.Complex64:
		v.SetComplex(complex(
			float64(math.Float32frombits(d.read32(f))),
			float64(math.Float32frombits(d.read32(f))),
		))
	case reflect.Complex128:
		v.SetComplex(complex(
			math.Float64frombits(d.read64(f)),
			math.Float64frombits(d.read64(f)),
		))
	}

	if f.SIndex != -1 {
		sv := struc.Field(f.SIndex)
		l := len(sfields)
		for i := 0; i < l; i++ {
			if sfields[i].Index != f.SIndex {
				continue
			}

			sf := sfields[i]
			sl := 0

			// Must use different codepath for signed/unsigned.
			switch f.DefType.Kind() {
			case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				sl = int(v.Int())
			case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				sl = int(v.Uint())
			default:
				panic(fmt.Errorf("unsupported sizeof type %s", f.DefType.String()))
			}

			// Strings are immutable, but we make a blank one so that we can
			// figure out the size later. It might be better to do something
			// more hackish, like writing the length into the string...
			switch sf.DefType.Kind() {
			case reflect.Slice:
				sv.Set(reflect.MakeSlice(sf.Type, sl, sl))
			case reflect.String:
				sv.SetString(string(make([]byte, sl)))
			default:
				panic(fmt.Errorf("unsupported sizeof target %s", sf.DefType.String()))
			}
		}
	}
}
