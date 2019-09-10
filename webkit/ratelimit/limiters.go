package ratelimit

type Limiters struct {
	Map map[interface{}]*Limiter
}

type RateMap map[interface{}][]Rate

func NewLimiters(rateMap RateMap) Limiters {
	mp := make(map[interface{}]*Limiter)
	for k, rates := range rateMap {
		mp[k] = NewLimiter(rates)
	}
	return Limiters{mp}
}


// Reached returns true if rate was exceeded.
func (ls Limiters) Reached(key interface{}) bool {
   limiter, ok := ls.Map[key]
   if !ok {
   	// no limiting in case of key does not exist
   	return false
   }
   return limiter.Reached()
}

// Check rate changed or not.
// If ErrorRateMismatch returned, just create a new limiter by New.
func (ls Limiters) ReachedRates(rateMap RateMap, key interface{}) (bool, error) {
	rates, ok := rateMap[key]
	if !ok {
		// no limiting in case of key does not exist
		return false, nil
	}

	limiter, ok := ls.Map[key]
	if !ok {
		return false, ErrorRateMismatch
	}

	return limiter.ReachedRates(rates)
}
