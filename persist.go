package elect

import (
	"bytes"
	"encoding/gob"
	"io/ioutil"
)

// PersistentStateToBytes converts PersistentState to []byte
// that is encoded by gob
// Parameters :
//  pState: State to be encoded
func PersistentStateToBytes(pState *PersistentState) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(pState)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), err
}

// BytesToPersistentState converts gob encoded bytes to PersistentState
func BytesToPersistentState(gobbed []byte) (*PersistentState, error) {
	buf := bytes.NewBuffer(gobbed)
	dec := gob.NewDecoder(buf)
	var ungobbed PersistentState
	err := dec.Decode(&ungobbed)
	return &ungobbed, err
}

// file mode that allows read and write for current process
// and read only for others
const UserReadWriteMode = 0644

// TODO: Move this to the actual place that does the write
func Write(filePath string, data []byte) error {
	return ioutil.WriteFile(filePath, data, UserReadWriteMode)
}

// ReadPersistentState reads PersistentState from the storage
// parameters :
//   filePath: Path to file from which the state should be read
func ReadPersistentState(filePath string) (*PersistentState, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	persistentState, err := BytesToPersistentState(data)
	return persistentState, err
}
