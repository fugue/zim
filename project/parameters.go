// Copyright 2020 Fugue, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package project

import "fmt"

func typeCheckParameter(typeName string, value interface{}) error {
	switch typeName {
	case "integer":
		if _, ok := value.(int); !ok {
			return fmt.Errorf("expected an integer; got %v", value)
		}
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected a string; got %v", value)
		}
	case "float":
		if _, ok := value.(float64); !ok {
			if _, ok := value.(int); !ok {
				return fmt.Errorf("expected a float; got %v", value)
			}
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected a boolean; got %v", value)
		}
	default:
		return fmt.Errorf("unknown parameter type: %s", typeName)
	}
	return nil
}
