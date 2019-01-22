/*
Copyright 2014 SAP SE

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

package sqltrace_test

import (
	"github.com/SAP/go-hdb/driver/sqltrace"
)

func Example() {
	sqltrace.SetOn(true)  //set SQL trace output active
	sqltrace.SetOn(false) //set SQL trace output inactive
}
