package web

// extraRoutes lets feature files register routes from init() without editing
// routes(), keeping feature work in separate files.
var extraRoutes []func(*server)
