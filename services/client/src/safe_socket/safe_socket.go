package safe_socket

import "io"

func SendAll(socket io.Writer, bytes []byte) error {
	written := 0

	for written < len(bytes) {
		n, err := socket.Write(bytes[written:])
		written += n

		if err != nil {
			return err
		}
	}
	return nil
}

func RecvAll(socket io.Reader, size int) ([]byte, error) {
	leftToRead := size
	buffTotal := make([]byte, 0, size)

	for leftToRead > 0 {
		buff := make([]byte, leftToRead)
		n, err := socket.Read(buff)
		leftToRead -= n
		buffTotal = append(buffTotal, buff[:n]...)

		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
	}

	return buffTotal, nil
}
