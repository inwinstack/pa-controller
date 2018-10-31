/*
Copyright © 2018 inwinSTACK.inc

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

package constants

const (
	// DefaultInternetPool is the default of internet pool name
	DefaultInternetPool = "internet"
	// AnnKeyAllowSecurity will set in Service resource to allow the security policy
	AnnKeyAllowSecurity = "inwinstack.com/allow-security-policy"
	// AnnKeyAllowNAT will set in Service resource to allow the nat policy
	AnnKeyAllowNAT = "inwinstack.com/allow-nat-policy"
	// AnnKeyExternalPool will set in Service resource to get the pool from this value
	AnnKeyExternalPool = "inwinstack.com/external-pool"
	// AnnKeyPublicIP will set in Service resource to show the public IP
	AnnKeyPublicIP = "inwinstack.com/allocated-public-ip"
	// AnnKeyServiceRefresh set in Service to refresh the annotations
	AnnKeyServiceRefresh = "inwinstack.com/service-refresh"
	// AnnKeyPolicyRetry set in NAT and Security to retry failed resource
	AnnKeyPolicyRetry = "inwinstack.com/policy-retry"
)
