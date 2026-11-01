// Package client queries and authenticates against the BIT SRUN campus gateway.
// It uses only the standard library and does not store credentials.
//
// Create a [Client] with [New] and [Config]. Each operation accepts a context
// for cancellation or a deadline covering the whole operation, which may make
// several HTTP requests.
//
// [Client.Login] and [Client.Logout] return a [PortalResponse]. Gateway rejection
// also returns a [GatewayError], which callers can inspect with [errors.As].
// Session changes may take effect after the response; query [Client.Status]
// to confirm them.
package client
