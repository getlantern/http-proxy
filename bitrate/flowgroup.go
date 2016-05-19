package bitrate

import "github.com/mxk/go-flowrate/flowrate"

type FlowGroupOptions struct {
	RateLimit      int64 // shared rate limit per group
	Utilization    float64
	MaxAttenuation float64
}

type FlowGroup struct {
	flows              flowsMap
	options            *FlowGroupOptions
	attenuationSteps   []float64
	prevSpareBandwidth int64
}

type flowsMap map[*flowrate.Limiter]flowData

type flowData struct {
	attenuationIndex int
}

func NewFlowGroup(opts *FlowGroupOptions, ls ...*flowrate.Limiter) *FlowGroup {
	if opts.RateLimit == 0 {
		panic("RateLimit should be set in FlowGroupOptions")
	}
	// Keep utilization always below 0.95 to both avoid convergence to zero and too many
	// attenuationSteps
	if opts.Utilization <= 0.0 || opts.Utilization >= 0.95 {
		panic("Utilization should be in the interval (0.0, 0.95)")
	}

	attenuationSteps := make([]float64, int(1.0/(1.0-opts.Utilization)))
	utilizationInverse := 1.0 - opts.Utilization
	// For example, with Utilization = 0.8, we would obtain [1.0 .8 .6 .4 .2]
	for i := range attenuationSteps {
		attenuationSteps[i] = 1.0 - utilizationInverse*float64(i)
	}

	return &FlowGroup{
		flows:            make(flowsMap),
		options:          opts,
		attenuationSteps: attenuationSteps,
	}
}

func (f *FlowGroup) AddLimiter(l *flowrate.Limiter) {
	f.flows[l] = flowData{attenuationIndex: 0}
}

func (f *FlowGroup) RemoveLimiter(l *flowrate.Limiter) (isDone bool) {
	delete(f.flows, l)
	return len(f.flows) == 0
}

func (f *FlowGroup) RebalanceLimits() {
	nFlows := len(f.flows)
	fairCommonLimit := float64(f.options.RateLimit) / float64(nFlows)

	// First pass will find underutilized streams and calculate attenuations
	spareBandwidth := int64(0)
	for lPtr, fData := range f.flows {
		status := (*lPtr).Status()
		attenuation := f.attenuationSteps[fData.attenuationIndex]
		flowLimit := int64(fairCommonLimit * attenuation)

		if status.CurRate < flowLimit {
			// If utilization is low, increase attenuation if possible
			if fData.attenuationIndex < len(f.attenuationSteps)-1 &&
				f.attenuationSteps[fData.attenuationIndex+1] < f.options.MaxAttenuation {
				fData.attenuationIndex++
			}
		} else {
			// If utilization is high, reduce attenuation if possible
			if fData.attenuationIndex != 0 {
				fData.attenuationIndex--
				// TODO l.SetLimit(commonLimit)
			}
		}

		// If the stream is attenuated, add the spare bandwidth
		if fData.attenuationIndex != 0 {
			spareBandwidth += int64(fairCommonLimit) - flowLimit
		}
	}

	// Second pass will assign spare bandwidth evenly and set limits
	adjustedGlobalRateLimit := f.options.RateLimit + spareBandwidth
	adjustedCommonLimit := int64(float64(adjustedGlobalRateLimit) / float64(nFlows))

	for lPtr, fData := range f.flows {
		if fData.attenuationIndex == 0 {
			(*lPtr).SetLimit(adjustedCommonLimit)
		} else {
			attenuation := f.attenuationSteps[fData.attenuationIndex]
			(*lPtr).SetLimit(int64(fairCommonLimit * attenuation))
		}
	}
}
