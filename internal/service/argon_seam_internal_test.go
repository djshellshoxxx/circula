package service

// ObserveArgon2 calls fn with the parameters of every Argon2id evaluation
// until the returned stop function runs. The service tests do not run in
// parallel, so one observer at a time is enough (BR-05 work-equality tests).
func ObserveArgon2(fn func(Argon2Params)) (stop func()) {
	argonObserver.Store(&fn)
	return func() { argonObserver.Store(nil) }
}
