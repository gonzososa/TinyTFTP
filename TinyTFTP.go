package main

import "fmt"
import "net"
import "os"
import "bytes"

const BLOCK_SIZE = 512

const (
  OCTET          = "octet"
  NETASCII       = "netascii"
)

const (
  RRQ uint16 = 1  + iota
  WRQ
  DATA
  ACK
  ERROR
)

var ERRORS = [...] string {
      "Not defined, see error message (if any).",
      "File not found.",
      "Access violation.",
      "Disk full or allocation exceeded.",
      "Illegal TFTP operation.",
      "Unknown transfer ID.",
      "File already exists.",
      "No such user.",
}

type Client struct {
  TID   *net.UDPAddr
  Conn  *net.UDPConn
  File  string
  Mode  string
}

func (c *Client) SendBytes (data []byte) error {
  _, err := c.Conn.WriteToUDP (data, c.TID)

  if err != nil {
    return err
  }

  return nil
}

func (c *Client) ReadBytes (buffer []byte) ([]byte, error) {
  _, _, err := c.Conn.ReadFromUDP (buffer)

  if err != nil {
    return nil, err
  }

  return buffer, nil
}

func Bytes2UInt16 (value []byte) uint16 {
  var a uint16 = uint16 (value [0] & 0xff) << 8
  var b uint16 = uint16 (value [1] & 0xff)
  return a + b
}

func Int2Bytes (value uint16) (retVal []byte) {
  retVal = []byte {byte (value >> 8), byte (value & 0xff)}
  return
}

func HandleRRQ (client *Client) {
  if _, err := os.Stat (client.File); err != nil {
    // file not found, send error message
    //opcode
    var data, message []byte
    data = append (data, append(Int2Bytes(5), Int2Bytes(1)...)...)
    //message
    message = []byte (ERRORS [1] + "\x00")

    data = append (data, message...)
    if err = client.SendBytes (data); err != nil {
      fmt.Println (err)
    }

    return
  }

  switch client.Mode {
    case OCTET:
      RRQBinary (client)
      break
    case NETASCII:
      panic ("Not supported!")
      break
  }
}

func RRQBinary (client *Client) {
  file, err := os.Open (client.File)
  if err != nil {
    fmt.Println ("Error opening file: ", client.File)
    fmt.Println (err)
    return
  }
  defer file.Close ()

  fileInfo, err := file.Stat ()
  if err != nil {
    fmt.Println ("Error gathering file information: ", client.File)
    fmt.Println (err)
    return
  }

  //var sendLast bool = false
  var fileSize int64 = fileInfo.Size ()
  var count uint16 = 1
  var fileBuffer []byte = make ([]byte, BLOCK_SIZE)

  var blockCount uint16 = uint16 (int (fileSize) / len (fileBuffer))
  if int (fileSize) % len (fileBuffer) != 0 {
    blockCount += 1
  }

  var data   []byte
  var buffer []byte = make ([]byte, 4)

  for {
    bytesRead, err := file.Read (fileBuffer)
    if err != nil {
      fmt.Println ("Error reading data from file: ", client.File)
      fmt.Println (err)
      return
    }

    //opcode
    data = Int2Bytes (3)
    //block number
    data = append (data, Int2Bytes(count)...)
    //data
    data = append (data, fileBuffer[:bytesRead]...)

    err = client.SendBytes (data)
    if err != nil {
      fmt.Println ("Network error while sending data to ", client.TID)
      fmt.Println (err)
      break
    }

    client.ReadBytes (buffer)
    /*_, _, err = conn.ReadFromUDP (buffer)*/
    /*if err != nil {
    fmt.Println (err)
    break
    }*/

    if (count == blockCount) {
      break
    }

    count++
  }
}

func RRQASCII (client *Client) {

}

func main () {
  port := ":69"

  udpAddress, err := net.ResolveUDPAddr ("udp4", port)
  if err != nil {
    fmt.Println ("Error resolving UDP address in ", port)
    fmt.Println (err)
    return
  }

  conn, err := net.ListenUDP ("udp", udpAddress)
  if err != nil {
    fmt.Println ("Error listing on udp port")
    fmt.Println (err)
    return
  }
  defer conn.Close ()

  var buffer = make ([]byte, 128)

  bytesRead, tid, err := conn.ReadFromUDP (buffer)
  if err != nil {
    fmt.Println ("Error reading data from connection")
    fmt.Println (err)
    return
  }

  if bytesRead > 0 {
    var opcode uint16 = Bytes2UInt16 (buffer [0:2])

    switch opcode {
      case RRQ: //RRQ
        //split filename & transfer mode
        var foo = bytes.Split (buffer [2:], []byte {0x00})

        client := &Client {
          Conn: conn,
          TID: tid,
          File: "/home/gonzalo/Atom/libffmpegsumo.so", //string (foo [0]),
          Mode: string (foo [1]),
        }

        HandleRRQ (client)
        break
      case WRQ: //WRQ
        //HandleWRQ (buffer, conn, remoteTID)
        break
    }
  }
}
