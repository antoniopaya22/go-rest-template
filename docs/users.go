package docs

// swagger:route GET /users users listUsers
//
// Lists users.
//
// This will show all available users by default.
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Schemes: http
//
//     Security:
//       api_key:
//       oauth: read, write
//
//     Responses:
//       default: genericError
//       200: someResponse
//       422: validationError