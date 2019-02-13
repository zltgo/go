package deepmind

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/zltgo/reflectx"
)

type Saver interface {
	Load() (*Model, error)
	LoadGraph() (*Model, error)
	LoadData() (map[string][]float64, error)
	Save(m *Model) error
}

type JsonSaver struct {
	reflectx.Reflector
	dir string
}

type LayerOpts struct {
	Names []string
	Opts  map[string]interface{}
}

func NewJsonSaver(dir string) JsonSaver {
	r := reflectx.NewReflector("", "", nil)
	r.Register(FCOpts{})
	r.Register(RNNOpts{})
	r.Register(LSTMOpts{})
	r.Register(GRUOpts{})
	r.Register(LayerOpts{})

	return JsonSaver{
		r,
		dir,
	}
}

func (j JsonSaver) Load() (m *Model, err error) {
	if m, err = j.LoadGraph(); err != nil {
		return nil, err
	}
	m.InitData, err = j.LoadData()
	return
}

func (j JsonSaver) LoadData() (map[string][]float64, error) {
	dataPath := filepath.Join(j.dir, "data.json")
	fp, err := os.OpenFile(dataPath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, err
	}
	defer fp.Close()

	dec := json.NewDecoder(fp)
	data := map[string][]float64{}
	if err = dec.Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

func (j JsonSaver) LoadGraph() (*Model, error) {
	graphPath := filepath.Join(j.dir, "graph.json")
	fp, err := os.OpenFile(graphPath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, err
	}
	defer fp.Close()

	b, err := ioutil.ReadAll(fp)
	if err != nil {
		return nil, err
	}

	//decode graph
	v, err := j.Decode(b)
	if err != nil {
		return nil, err
	}
	cfg, ok := v.(LayerOpts)
	if !ok {
		return nil, errors.Errorf("expected LayerOpts type from graph.json, got %T", v)
	}

	//create layers
	m := NewModel()
	m.Layers, err = NewLayers(cfg)
	return m, err
}

//create layers by LayerOpts
func NewLayers(cfg LayerOpts) ([]Layer, error) {
	layers := make([]Layer, len(cfg.Names))
	for i, name := range cfg.Names {
		opt, ok := cfg.Opts[name]
		if !ok {
			return nil, errors.Errorf("options of %s not found", name)
		}
		layer, err := newLayer(name, opt)
		if err != nil {
			return nil, err
		}
		layers[i] = layer
	}
	return layers, nil
}

func newLayer(name string, val interface{}) (Layer, error) {
	switch opt := val.(type) {
	case FCOpts:
		return NewFC(name, opt)
	case RNNOpts:
		return NewRNN(name, opt)
	case LSTMOpts:
		return NewLSTM(name, opt)
	case GRUOpts:
		return NewGRU(name, opt)
	default:
		return nil, errors.Errorf("newLayer: unknown options type: %T", val)
	}
}

func (j JsonSaver) Save(m *Model) error {
	dataPath := filepath.Join(j.dir, "data.json")
	graphPath := filepath.Join(j.dir, "graph.json")

	//rename old files.
	if err := renameWithDateTime(j.dir, "data.json"); err != nil {
		return errors.Wrap(err, "rename data.json")
	}
	if err := renameWithDateTime(j.dir, "graph.json"); err != nil {
		return errors.Wrap(err, "rename graph.json")
	}

	//save graph
	numLayers := len(m.Layers)
	gh := LayerOpts{
		Names: make([]string, numLayers),
		Opts:  make(map[string]interface{}, numLayers),
	}
	for i, layer := range m.Layers {
		gh.Names[i] = layer.Name()
		gh.Opts[layer.Name()] = layer.Options()
	}
	b, err := j.Encode(gh)
	if err != nil {
		return err
	}
	if err = saveFile(graphPath, b); err != nil {
		return errors.Wrap(err, "save graph")
	}

	//save data
	data := map[string][]float64{}
	for _, n := range m.Learnables() {
		data[n.Name()] = GetBackingF64(n)
	}

	if err = saveFile(dataPath, data); err != nil {
		return errors.Wrap(err, "save data")
	}

	return nil
}

func saveFile(path string, val interface{}) error {
	fp, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer fp.Close()

	if b, ok := val.([]byte); ok {
		_, err = fp.Write(b)
		return err
	}
	return json.NewEncoder(fp).Encode(val)
}

//Add date and time at the end of file name.
func renameWithDateTime(dir, name string) error {
	path := filepath.Join(dir, name)
	fi, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	//2006-01-02 15:04:05 is the birthday of golang?
	day := fi.ModTime().Format("2006-01-02")
	newDir := filepath.Join(dir, day)
	if err = os.MkdirAll(newDir, os.ModePerm); err != nil {
		return err
	}

	time := fi.ModTime().Format("(15:04:05)")
	return os.Rename(path, filepath.Join(newDir, name+time))
}
