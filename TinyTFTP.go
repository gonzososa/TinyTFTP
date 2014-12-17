package main

import "fmt"
import "net"
import "os"
import "bytes"

const BLOCK_SIZE = 512

func Bytes2UInt16 (value []byte) uint16 {
  var a uint16 = uint16 (value [0] & 0xff) << 8
  var b uint16 = uint16 (value [1] & 0xff)
  return a + b
}

func Int2Bytes (value uint16) (retVal []byte) {
  retVal = []byte {uint8 (value >> 8), uint8 (value & 0xff)}
  return
}

func HandleRRQ (buffer []byte, conn *net.UDPConn, tid *net.UDPAddr) {
  var foo = bytes.Split (buffer [2:], []byte {0x00})
  var filename string = string (foo [0])
  var mode string = string (foo [1])

  if _, err := os.Stat (filename); err == nil {
    var fileBuffer []byte

    if mode == "octet" {
      fileBuffer = make ([]byte, BLOCK_SIZE)
    } else if mode == "netascii" {
      panic ("Modo no soportado!!")
    } else {
      panic ("Modo desconocido!!")
    }

    file, err := os.Open (filename)
    if err != nil {
      fmt.Println ("Error al abrir el archivo: ", filename)
      fmt.Println (err)
      return
    }
    defer file.Close ()

    fileInfo, err := file.Stat ()
    if err != nil {
      fmt.Println ("Error al obtener la información del archivo: ", filename)
      fmt.Println (err)
      return
    }

    //var sendLast bool = false
    var fileLength int64 = fileInfo.Size ()
    var count uint16 = 1

    var blockCount uint16 = uint16 (int (fileLength) / len (fileBuffer))
    if int (fileLength) % len (fileBuffer) != 0 {
      blockCount += 1
    }

    for {
      n, err := file.Read (fileBuffer)
      if err != nil {
        fmt.Println (err)
        return
      }

      //opcode
      var data []byte = Int2Bytes (3)

      //block number
      data = append (data, Int2Bytes(count)...)

      //data
      switch mode {
        case "octet":
        data = append (data, fileBuffer[:n]...)
        break
        case "netascii":
        panic ("not supported...")
        break
      }

      _, err = conn.WriteToUDP (data, tid)
      if err != nil {
        fmt.Println (err)
        break
      }
      data = nil

      buffer = make ([]byte, 4)
      /*_, _, err =*/ conn.ReadFromUDP (buffer)
      /*if err != nil {
      fmt.Println (err)
      break
      }*/

      if (count == blockCount) {
        break
      }

      count++
    }

    } else {
      // file not found, send error message
      // opcode + error code
      var data []byte
      data = append (data, append(Int2Bytes(5), Int2Bytes(1)...)...)
      // message
      message := []byte ("File not found.\x00")
      data = append (data, message...)

      _, err := conn.WriteToUDP (data, tid)
      if err != nil {
        fmt.Println (err)
      }
    }
}

func HandleWRQ (buffer []byte, conn *net.UDPConn, tid *net.UDPAddr) {
  //filename & mode
  var foo = bytes.Split (buffer [2:], []byte {0x00})
  var filename string = string (foo [0])
  //var mode string = string (foo [1])

  file, err := os.Create (filename)
  if err != nil {
    fmt.Println (err)
    return
  }
  defer file.Close ()

  data := []byte {0x00, 0x03, 0x00, 0x00}
  _, err = conn.WriteToUDP (data, tid)
  if err != nil {
    fmt.Println (err)
    return
  }

  data = make ([]byte, BLOCK_SIZE)

  for {
    n, remoteTID, err := conn.ReadFromUDP (data)
    if err != nil {
      fmt.Println (err)
      break
    }

    // extraer el opcode y block # de data
    // var opcode2 = data[0:2]
    // var block uint16 = data [2:4]
    //
    _,  err = file.Write (data[4:n])
    if err != nil {
      fmt.Println (err)
      break
    }

    data = []byte {0x00, 0x03}
    // agregar block # recibido
    // data = append (data, ...)
    _, err = conn.WriteToUDP (data, remoteTID)
    if err != nil {
      fmt.Println (err)
      break
    }

    if n < BLOCK_SIZE {
      break
    }
  }
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

  var buffer = make ([]byte, BLOCK_SIZE)

  bytesRead, remoteTID, err := conn.ReadFromUDP (buffer)
  if err != nil {
    fmt.Println ("Error reading data from connection")
    fmt.Println (err)
    return
  }

  if bytesRead > 0 {
    var opcode uint16 = Bytes2UInt16 (buffer [0:2])

    switch opcode {
      case 1: //RRQ
        HandleRRQ (buffer, conn, remoteTID)
        break
      case 2: //WRQ
        HandleWRQ (buffer, conn, remoteTID)
        break
    }
  }
}
