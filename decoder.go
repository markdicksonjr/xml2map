package xml2map

import (
	"encoding/xml"
	"io"
	"strings"
	"github.com/pkg/errors"
)

const (
	AttrPrefix = "@"
	TextPrefix = "#text"
)

type node struct {
	Parent *node
	Value map[string]interface{}
	Attrs []xml.Attr
	Label string
	Text string
	HasMany bool
}

type Decoder struct {
	r io.Reader
}

func NewDecoder(reader io.Reader) *Decoder {
	return &Decoder{r: reader}
}

func (d *Decoder) Decode() (map[string]interface{}, error) {
	decoder := xml.NewDecoder(d.r)
	n := &node{}
	stack := make([]*node, 0)

	for {
		token, err := decoder.Token()
		if err != nil && err != io.EOF {
			return nil, err
		}

		if token == nil {
			break
		}

		switch tok := token.(type) {
		case xml.StartElement:
			{
				n = &node{
					Label:  tok.Name.Local,
					Parent: n,
					Value:  map[string]interface{}{tok.Name.Local:map[string]interface{}{}},
					Attrs:  tok.Attr,
				}

				if len(tok.Attr) > 0 {
					m := make(map[string]interface{})
					for _, attr := range tok.Attr {
						if len(attr.Name.Space) > 0 {
							m[AttrPrefix+attr.Name.Space+":"+attr.Name.Local] = attr.Value
						} else {
							m[AttrPrefix+attr.Name.Local] = attr.Value
						}
					}
					n.Value[tok.Name.Local] = m
				}
				stack = append(stack, n)

				if n.Parent != nil {
					n.Parent.HasMany = true
				}
			}

		case xml.CharData:
			stack[len(stack)-1].Text = strings.TrimSpace(string(tok))

		case xml.EndElement:
			{
				length := len(stack)
				if length == 1 {
					return stack[0].Value, nil
				}

				stack, n = stack[:length-1], stack[length-1]

				if !n.HasMany {
					if len(n.Attrs) > 0 {
						m := n.Value[n.Label].(map[string]interface{})
						m[TextPrefix] = n.Text
					} else {
						n.Value[n.Label] = n.Text
					}
				}

				if v, ok := n.Parent.Value[n.Parent.Label]; ok {
					m := v.(map[string]interface{})
					if v, ok = m[n.Label]; ok {
						switch item := v.(type) {
						case string:
							m[n.Label] = []string{item, n.Value[n.Label].(string)}
						case []string:
							m[n.Label] = append(item, n.Value[n.Label].(string))
						case map[string]interface{}:
							vm := getMap(n)
							if vm != nil {
								m[n.Label] = []map[string]interface{}{item, vm}
							}
						case []map[string]interface{}:
							vm := getMap(n)
							if vm != nil {
								m[n.Label] = append(item, vm)
							}
						}
					} else {
						m[n.Label] = n.Value[n.Label]
					}

				} else {
					n.Parent.Value[n.Parent.Label] = n.Value[n.Label]
				}
				n = n.Parent
			}
		}
	}

	return nil, errors.New("Invalid document")
}

func getMap(node *node) map[string]interface{} {
	if v, ok := node.Value[node.Label]; ok {
		switch v.(type) {
		case string:
			return map[string]interface{}{node.Label: v}
		case map[string]interface{}:
			return node.Value[node.Label].(map[string]interface{})
		}
	}

	return nil
}