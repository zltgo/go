package api

// Middware acts as a list of Handler handler.
// Middware is effectively immutable:
// once created, it will always hold
// the same set of handler in the same order.
type Middware struct {
	handlers []Handler
}

// New creates a new chain,
// memorizing the given list of handler handler.
// New serves no other function,
// handler are only called upon a call to Then().
func NewMiddware(handlers ...Handler) Middware {
	return Middware{append(([]Handler)(nil), handlers...)}
}

// Then chains the handler and returns the final Handler.
//     New(m1, m2, m3).Then(h)
// is equivalent to:
//     []handler{m1,m2,m3,h}
// When the request comes in, it will be passed to m1, then m2, then m3
// and finally, the given handler
// (assuming every handler calls the following one).
//
// A chain can be safely reused by calling Then() several times.
//     stdStack := alice.New(ratelimitHandler, csrfHandler)
//     indexPipe = stdStack.Then(indexHandler)
//     authPipe = stdStack.Then(authHandler)
// Note that handler are called on every call to Then()
// and thus several instances of the same handler will be created
// when a chain is reused in this way.
// For proper handler, this should cause no problems.
// Input parameter can be empty but can't be nil interface.
func (c Middware) Then(fn ...interface{}) []Handler {
	rv := make([]Handler, len(c.handlers)+len(fn))
	copy(rv, c.handlers)

	for i := 0; i < len(fn); i++ {
		rv[i+len(c.handlers)] = H(fn[i])
	}

	return rv
}

// Append extends a chain, adding the specified handler
// as the last ones in the request flow.
//
// Append returns a new chain, leaving the original one untouched.
//
//     stdChain := alice.New(m1, m2)
//     extChain := stdChain.Append(m3, m4)
// requests in stdChain go m1 -> m2
//  requests in extChain go m1 -> m2 -> m3 -> m4
func (c Middware) Append(handlers ...Handler) Middware {
	return Middware{append(c.handlers, handlers...)}
}

// Extend extends a chain by adding the specified chain
// as the last one in the request flow.
//
// Extend returns a new chain, leaving the original one untouched.
//
//     stdChain := alice.New(m1, m2)
//     ext1Chain := alice.New(m3, m4)
//     ext2Chain := stdChain.Extend(ext1Chain)
//  Requests in stdChain go  m1 -> m2
//  Requests in ext1Chain go m3 -> m4
//  Requests in ext2Chain go m1 -> m2 -> m3 -> m4
//
// Another example:
//  aHtmlAfterNosurf := alice.New(m2)
// 	aHtml := alice.New(m1, func(h Handler) Handler {
// 		csrf := nosurf.New(h)
// 		csrf.SetFailureHandler(aHtmlAfterNosurf.ThenFunc(csrfFail))
// 		return csrf
// 	}).Extend(aHtmlAfterNosurf)
//	Requests to aHtml hitting nosurfs success handler go m1 -> nosurf -> m2 -> target-handler
//	Rrequests to aHtml hitting nosurfs failure handler go m1 -> nosurf -> m2 -> csrfFail
func (c Middware) Extend(m Middware) Middware {
	return c.Append(m.handlers...)
}
