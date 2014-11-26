package processor

import (
	"github.com/Hranoprovod/shared"
	"github.com/Hranoprovod/accumulator"
	"github.com/Hranoprovod/reporter"
	"regexp"
	"sort"
	"time"
)

const (
	dateBeginning = 0
	dateEnd       = 1
)

type ProcessorOptions struct {
	DateFormat string
	HasBeginning bool
	HasEnd bool
	BeginningTime time.Time
	EndTime time.Time
	Unresolved bool
	SingleElement string
	SingleFood string
	Totals bool
}

// Processor contains the processor data
type Processor struct {
	options  *ProcessorOptions
	db       *shared.NodeList
	reporter *reporter.Reporter
}

func DefaultProcessorOptions() *ProcessorOptions {
	return &ProcessorOptions{
		"2006/01/02",
		false,
		false,
		time.Now(),
		time.Now(),
		false,
		"",
		"",
		true,
	}
}

// NewProcessor creates new node processor
func NewProcessor(options *ProcessorOptions, db *shared.NodeList, reporter *reporter.Reporter) *Processor {
	return &Processor{
		options,
		db,
		reporter,
	}
}

func isGoodDate(time, compareTime time.Time, compareType int) bool {
	if time.Equal(compareTime) {
		return true
	}
	if compareType == dateBeginning {
		return time.After(compareTime)
	}
	return time.Before(compareTime)
}

func (p *Processor) Process(ln *shared.LogNode) error {
	if (p.options.HasBeginning && !isGoodDate(ln.Time, p.options.BeginningTime, dateBeginning)) || (p.options.HasEnd && !isGoodDate(ln.Time, p.options.EndTime, dateEnd)) {
		return nil
	}

	if p.options.Unresolved {
		return p.unresolvedProcessor(ln)
	}
	if len(p.options.SingleElement) > 0 {
		return p.singleProcessor(ln)
	}
	if len(p.options.SingleFood) > 0 {
		return p.singleFoodProcessor(ln)
	}
	return p.defaultProcessor(ln)

}

func (p *Processor) unresolvedProcessor(ln *shared.LogNode) error {
	for _, e := range *ln.Elements {
		_, found := (*p.db)[e.Name]
		if !found {
			p.reporter.PrintUnresolvedRow(e.Name)
		}
	}
	return nil
}

func (p *Processor) singleProcessor(ln *shared.LogNode) error {
	acc := accumulator.NewAccumulator()
	singleElement := p.options.SingleElement
	for _, e := range *ln.Elements {
		repl, found := (*p.db)[e.Name]
		if found {
			for _, repl := range *repl.Elements {
				if repl.Name == singleElement {
					acc.Add(repl.Name, repl.Val*e.Val)
				}
			}
		} else {
			if e.Name == singleElement {
				acc.Add(e.Name, e.Val)
			}
		}
	}
	if len(*acc) > 0 {
		arr := (*acc)[singleElement]
		p.reporter.PrintSingleElementRow(ln.Time, p.options.SingleElement, arr[accumulator.Positive], arr[accumulator.Negative])
	}
	return nil
}

func (p *Processor) singleFoodProcessor(ln *shared.LogNode) error {
	for _, e := range *ln.Elements {
		matched, err := regexp.MatchString(p.options.SingleFood, e.Name)
		if err != nil {
			return err
		}
		if matched {
			p.reporter.PrintSingleFoodRow(ln.Time, e.Name, e.Val)
		}
	}
	return nil
}

func (p *Processor) defaultProcessor(ln *shared.LogNode) error {
	acc := accumulator.NewAccumulator()
	p.reporter.PrintDate(ln.Time)
	for _, element := range *ln.Elements {
		p.reporter.PrintElement(element)
		if repl, found := (*p.db)[element.Name]; found {
			for _, repl := range *repl.Elements {
				res := repl.Val * element.Val
				p.reporter.PrintIngredient(repl.Name, res)
				acc.Add(repl.Name, res)
			}
		} else {
			p.reporter.PrintIngredient(element.Name, element.Val)
			acc.Add(element.Name, element.Val)
		}
	}
	if p.options.Totals {
		var ss sort.StringSlice
		if len(*acc) > 0 {
			p.reporter.PrintTotalHeader()
			for name := range *acc {
				ss = append(ss, name)
			}
			sort.Sort(ss)
			for _, name := range ss {
				arr := (*acc)[name]
				p.reporter.PrintTotalRow(name, arr[accumulator.Positive], arr[accumulator.Negative])
			}
		}
	}
	return nil
}