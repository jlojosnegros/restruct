package restruct

import (
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
)

// Packer is a type capable of packing a native value into a binary
// representation. The Pack function is expected to overwrite a number of
// bytes in buf then return a slice of the remaining buffer. Note that you
// must also implement SizeOf, and returning an incorrect SizeOf will cause
// the encoder to crash. The SizeOf should be equal to the number of bytes
// consumed from the buffer slice in Pack. You may use a pointer receiver even
// if the type is used by value.
type Packer interface {
	Sizer
	Pack(buf []byte, order binary.ByteOrder) ([]byte, error)
}

type encoder struct {
	order      binary.ByteOrder
	buf        []byte
	struc      reflect.Value
	sfields    []field
	bitCounter uint8
	realSize   uint64
	initBuf    []byte
}

func (e *encoder) assignBuffer(in []byte) {
	e.buf = in
	e.initBuf = in
}

func (e *encoder) writeBits(f field, inBuf []byte) {
	// hacemos la salida del tama;o esperado por el cliente
	defer func() {
		fmt.Printf("currentOut %#v\n", e.initBuf)
	}()
	fmt.Println("======writeBits=====")
	fmt.Printf("inputSize: %#v\n", len(inBuf))
	fmt.Printf("inBuf: %#v\n", inBuf)

	if f.BitSize == 0 {
		f.BitSize = uint8(f.Type.Bits())
	}
	fmt.Printf("bitCount: %#v\n", e.bitCounter)
	fmt.Printf("bitSize: %#v\n", f.BitSize)

	// Case A: There is no bitfields
	if (e.bitCounter + (f.BitSize % 8)) == 0 {
		fmt.Println("CASE A")
		copy(e.buf, inBuf)

		e.buf = e.buf[len(inBuf):]

		return
	}

	if (e.bitCounter + f.BitSize) <= 8 {
		fmt.Println("CASE B")
		fmt.Printf("in: 0x%X\n", inBuf[0])
		var mask uint8 = (0x01 << f.BitSize) - 1
		fmt.Printf("mask: 0x%X\n", mask)
		fmt.Printf("in shifted: 0x%X\n", inBuf[0]>>e.bitCounter)

		e.buf[0] |= (inBuf[0] & mask) << (8 - (e.bitCounter + f.BitSize))

		fmt.Printf("out[0]: 0x%X\n", e.buf[0])
		e.bitCounter = (e.bitCounter + f.BitSize) % 8
		fmt.Printf("bitCount(new): 0x%X\n", e.bitCounter)

		return
	}

	fmt.Println("CASE C")
	nbytes := (e.bitCounter + f.BitSize) / 8
	nbits := (e.bitCounter + f.BitSize) % 8

	fmt.Printf("nbytes: %#v\n", nbytes)
	fmt.Printf("nbits: %#v\n", nbits)

	var shift uint8 = e.bitCounter
	fmt.Printf("shift: %#v\n", shift)

	// var maskShift uint8 = 8 - shift
	// fmt.Printf("maskShift: %#v\n", maskShift)

	var carryShift uint8 = (8 - shift)
	fmt.Printf("carryShift: %#v\n", carryShift)

	var carryMask uint8 = (0x01 << shift) - 1
	fmt.Printf("carryMask: 0x%X\n", carryMask)
	//	var mask uint8 = (0x01 << maskShift) - 1
	//	fmt.Printf("mask: 0x%X\n", mask)

	var carry uint8 = 0
	fmt.Println("bucle")
	var outIdx uint8 = 0
	for ; outIdx < nbytes; outIdx++ {
		fmt.Printf("in[%d]: 0x%X\n", outIdx, inBuf[outIdx])
		fmt.Printf("carry: 0x%X\n", carry)
		fmt.Printf("out[%d]: 0x%X\n", outIdx, e.buf[outIdx])
		e.buf[outIdx] |= (inBuf[outIdx] >> shift) | carry
		fmt.Printf("out[%d]: 0x%X\n", outIdx, e.buf[outIdx])
		carry = (inBuf[outIdx] & carryMask) << carryShift
		fmt.Printf("new carry: 0x%X\n", carry)
	}
	fmt.Println("fin bucle: ", outIdx)

	if nbits > 0 {
		e.buf[outIdx] = carry
		fmt.Printf("out[%d]: 0x%X\n", outIdx, e.buf[outIdx])
	}

	// update buffer
	fmt.Printf("outbuf: %#v\n", e.buf)
	e.buf = e.buf[nbytes:]
	fmt.Printf("outbuf: %#v\n", e.buf)
	e.bitCounter = nbits
	return
}

func (e *encoder) write8(f field, x uint8) {
	typeSize := uint8(reflect.TypeOf(x).Size())

	b := make([]byte, typeSize)
	b[0] = x
	e.writeBits(f, b)
	// e.buf[0] = x
	// e.buf = e.buf[1:]
}

func (e *encoder) write16(f field, x uint16) {
	typeSize := uint8(reflect.TypeOf(x).Size())

	b := make([]byte, typeSize)
	e.order.PutUint16(b[0:typeSize], x)

	e.writeBits(f, b)

	// e.order.PutUint16(e.buf[0:2], x)
	// e.buf = e.buf[2:]
}

func (e *encoder) write32(f field, x uint32) {
	typeSize := uint8(reflect.TypeOf(x).Size())

	b := make([]byte, typeSize)
	e.order.PutUint32(b[0:typeSize], x)

	e.writeBits(f, b)

	// e.order.PutUint32(e.buf[0:4], x)
	// e.buf = e.buf[4:]
}

func (e *encoder) write64(f field, x uint64) {
	typeSize := uint8(reflect.TypeOf(x).Size())

	b := make([]byte, typeSize)
	e.order.PutUint64(b[0:typeSize], x)

	e.writeBits(f, b)

	// e.order.PutUint64(e.buf[0:8], x)
	// e.buf = e.buf[8:]
}

func (e *encoder) writeS8(f field, x int8) { e.write8(f, uint8(x)) }

func (e *encoder) writeS16(f field, x int16) { e.write16(f, uint16(x)) }

func (e *encoder) writeS32(f field, x int32) { e.write32(f, uint32(x)) }

func (e *encoder) writeS64(f field, x int64) { e.write64(f, uint64(x)) }

func (e *encoder) skipn(count int) {
	e.buf = e.buf[count:]
}

func (e *encoder) skip(f field, v reflect.Value) {
	e.skipn(f.SizeOf(v))
}

func (e *encoder) packer(v reflect.Value) (Packer, bool) {
	if s, ok := v.Interface().(Packer); ok {
		return s, true
	}

	if !v.CanAddr() {
		return nil, false
	}

	if s, ok := v.Addr().Interface().(Packer); ok {
		return s, true
	}

	return nil, false
}

func (e *encoder) write(f field, v reflect.Value) {
	if f.Name != "_" {
		if s, ok := e.packer(v); ok {
			var err error
			e.buf, err = s.Pack(e.buf, e.order)
			if err != nil {
				panic(err)
			}
			return
		}
	} else {
		e.skipn(f.SizeOf(v))
		return
	}

	struc := e.struc
	sfields := e.sfields
	order := e.order

	if f.Order != nil {
		e.order = f.Order
		defer func() { e.order = order }()
	}

	if f.Skip != 0 {
		e.skipn(f.Skip)
	}

	// If this is a sizeof field, pull the current slice length into it.
	if f.SIndex != -1 {
		sv := struc.Field(f.SIndex)

		switch f.DefType.Kind() {
		case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			v.SetInt(int64(sv.Len()))
		case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			v.SetUint(uint64(sv.Len()))
		default:
			panic(fmt.Errorf("unsupported sizeof type %s", f.DefType.String()))
		}
	}

	switch f.Type.Kind() {
	case reflect.Array, reflect.Slice, reflect.String:
		switch f.DefType.Kind() {
		case reflect.Array, reflect.Slice, reflect.String:
			ef := f.Elem()
			l := v.Len()
			for i := 0; i < l; i++ {
				e.write(ef, v.Index(i))
			}
		default:
			panic(fmt.Errorf("invalid array cast type: %s", f.DefType.String()))
		}

	case reflect.Struct:
		e.struc = v
		e.sfields = cachedFieldsFromStruct(f.Type)
		l := len(e.sfields)
		fmt.Println("22")
		// if e.realSize == 0 {
		// 	for idx, fld := range e.sfields {
		// 		if fld.BitSize == 0 {
		// 			e.realSize += uint64(fld.Type.Size()) * 8
		// 		} else {
		// 			e.realSize += uint64(fld.BitSize)
		// 		}
		// 		fmt.Println("idx:", idx, " realSize:", e.realSize)
		// 	}
		//
		// 	if (e.realSize % 8) != 0 {
		// 		e.realSize = 1 + (e.realSize / 8)
		// 	} else {
		// 		e.realSize = (e.realSize / 8)
		// 	}
		// 	if uint64(len(e.buf)) != e.realSize {
		// 		fmt.Println("Changing buffer len : from->to", len(e.buf), e.realSize)
		// 		e.assignBuffer(make([]byte, e.realSize))
		// 	}
		// }
		for i := 0; i < l; i++ {
			f := e.sfields[i]
			v := v.Field(f.Index)
			if v.CanSet() {
				e.write(f, v)
			} else {
				e.skip(f, v)
			}
		}
		e.sfields = sfields
		e.struc = struc

	case reflect.Int8:
		e.writeS8(f, int8(v.Int()))
	case reflect.Int16:
		e.writeS16(f, int16(v.Int()))
	case reflect.Int32:
		e.writeS32(f, int32(v.Int()))
	case reflect.Int64:
		e.writeS64(f, int64(v.Int()))

	case reflect.Uint8:
		e.write8(f, uint8(v.Uint()))
	case reflect.Uint16:
		e.write16(f, uint16(v.Uint()))
	case reflect.Uint32:
		e.write32(f, uint32(v.Uint()))
	case reflect.Uint64:
		e.write64(f, uint64(v.Uint()))

	case reflect.Float32:
		e.write32(f, math.Float32bits(float32(v.Float())))
	case reflect.Float64:
		e.write64(f, math.Float64bits(float64(v.Float())))

	case reflect.Complex64:
		x := v.Complex()
		e.write32(f, math.Float32bits(float32(real(x))))
		e.write32(f, math.Float32bits(float32(imag(x))))
	case reflect.Complex128:
		x := v.Complex()
		e.write64(f, math.Float64bits(float64(real(x))))
		e.write64(f, math.Float64bits(float64(imag(x))))
	}
}
