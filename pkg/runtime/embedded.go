/*
Copyright 2014 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package runtime

import (
	"errors"

	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/conversion"
)

type encodable struct {
	e        Encoder `json:"-"`
	obj      Object
	versions []unversioned.GroupVersion `json:"-"`
}

func (e encodable) GetObjectKind() unversioned.ObjectKind { return e.obj.GetObjectKind() }

// NewEncodable creates an object that will be encoded with the provided codec on demand.
// Provided as a convenience for test cases dealing with internal objects.
func NewEncodable(e Encoder, obj Object, versions ...unversioned.GroupVersion) Object {
	if _, ok := obj.(*Unknown); ok {
		return obj
	}
	return encodable{e, obj, versions}
}

func (re encodable) UnmarshalJSON(in []byte) error {
	return errors.New("runtime.encodable cannot be unmarshalled from JSON")
}

// Marshal may get called on pointers or values, so implement MarshalJSON on value.
// http://stackoverflow.com/questions/21390979/custom-marshaljson-never-gets-called-in-go
func (re encodable) MarshalJSON() ([]byte, error) {
	return Encode(re.e, re.obj)
}

// NewEncodableList creates an object that will be encoded with the provided codec on demand.
// Provided as a convenience for test cases dealing with internal objects.
func NewEncodableList(e Encoder, objects []Object, versions ...unversioned.GroupVersion) []Object {
	out := make([]Object, len(objects))
	for i := range objects {
		if _, ok := objects[i].(*Unknown); ok {
			out[i] = objects[i]
			continue
		}
		out[i] = NewEncodable(e, objects[i], versions...)
	}
	return out
}

func (re *Unknown) UnmarshalJSON(in []byte) error {
	if re == nil {
		return errors.New("runtime.Unknown: UnmarshalJSON on nil pointer")
	}
	re.TypeMeta = TypeMeta{}
	re.RawJSON = append(re.RawJSON[0:0], in...)
	return nil
}

// Marshal may get called on pointers or values, so implement MarshalJSON on value.
// http://stackoverflow.com/questions/21390979/custom-marshaljson-never-gets-called-in-go
func (re Unknown) MarshalJSON() ([]byte, error) {
	if re.RawJSON == nil {
		return []byte("null"), nil
	}
	return re.RawJSON, nil
}

func DefaultEmbeddedConversions() []interface{} {
	return []interface{}{
		func(in *Object, out *RawExtension, s conversion.Scope) error {
			if in == nil {
				out.RawJSON = []byte("null")
				return nil
			}
			obj := *in
			if unk, ok := obj.(*Unknown); ok {
				if unk.RawJSON != nil {
					out.RawJSON = unk.RawJSON
					return nil
				}
				obj = out.Object
			}
			if obj == nil {
				out.RawJSON = nil
				return nil
			}
			out.Object = obj
			return nil
		},

		func(in *RawExtension, out *Object, s conversion.Scope) error {
			if in.Object != nil {
				*out = in.Object
				return nil
			}
			data := in.RawJSON
			if len(data) == 0 || (len(data) == 4 && string(data) == "null") {
				*out = nil
				return nil
			}
			*out = &Unknown{
				RawJSON: data,
			}
			return nil
		},
	}
}