package util

import (
  "encoding/binary"
  "bytes"
  "strconv"
)

func bin(i int, prefix bool) string {
  i64 := int64(i)
  if prefix {
    return "0b" + strconv.FormatInt(i64, 2) // base 2 for binary
  }
  return strconv.FormatInt(i64, 2) // base 2 for binary
}

func bin2int(binStr string) int {
  // base 2 for binary
  result, _ := strconv.ParseInt(binStr, 2, 64)
  return int(result)
}

func oct(i int, prefix bool) string {
  i64 := int64(i)
  if prefix {
    return "0o" + strconv.FormatInt(i64, 8) // base 8 for octal
  }
  return strconv.FormatInt(i64, 8) // base 8 for octal
}

func oct2int(octStr string) int {
  // base 8 for octal
  result, _ := strconv.ParseInt(octStr, 8, 64)
  return int(result)
}

func hexm(i int64, prefix bool) string {
  if prefix {
    return "0x" + strconv.FormatInt(i, 16) // base 16 for hexadecimal
  }
  return strconv.FormatInt(i, 16) // base 16 for hexadecimal
}

func hex2int(hexStr string) int64 {
  // base 16 for hexadecimal
  result, _ := strconv.ParseUint(hexStr, 16, 64)
  return int64(result)
}

// Int2byte int32 to 4 bytes
func Int2byte(i uint64, byteSize int) []byte {
  buf := make([]byte, byteSize)
  switch byteSize {
  case 2:
    binary.BigEndian.PutUint16(buf, uint16(i))
  case 4:
    binary.BigEndian.PutUint32(buf, uint32(i))
  case 8:
    binary.BigEndian.PutUint64(buf, uint64(i))
  }
  return buf
}

// Str2fixedByte string to fixed lenght of bytes
func Str2fixedByte(varStr string, byteLen int) []byte {
  buf := make([]byte, byteLen)
  w := bytes.NewBuffer(buf)
  w.WriteString(varStr)
  return w.Bytes()
}
