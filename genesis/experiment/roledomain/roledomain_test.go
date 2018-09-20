/*
 *    Copyright 2018 Insolar
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package roledomain

// func TestRoleDomain_GetNodeRecord(t *testing.T) {
// 	roleDomain := NewRoleDomain()
// 	rRecord := rolerecord.NewRoleRecord("test", core.RoleHeavyExecutor)
// 	nodeRef := roleDomain.RegisterNode(rRecord.PublicKey, rRecord.Role)
//
// 	gotRoleRecord := roleDomain.GetNodeRecord(nodeRef)
// 	assert.NotNil(t, gotRoleRecord)
// 	assert.Equal(t, rRecord, gotRoleRecord)
// }
//
// func TestRoleDomain_RemoveNode(t *testing.T) {
//
// 	roleDomain := NewRoleDomain()
// 	rRecord := rolerecord.NewRoleRecord("test", core.RoleHeavyExecutor)
// 	nodeRef := roleDomain.RegisterNode(rRecord.PublicKey, rRecord.Role)
//
// 	gotRoleRecord := roleDomain.GetNodeRecord(nodeRef)
// 	assert.NotNil(t, gotRoleRecord)
// 	assert.Equal(t, rRecord, gotRoleRecord)
//
// 	roleDomain.RemoveNode(nodeRef)
// 	nothing := roleDomain.GetNodeRecord(nodeRef)
// 	assert.Nil(t, nothing)
// }
