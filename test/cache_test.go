package test

import (
	"github.com/florinutz/gh-recruiter/cache"
	"reflect"
	"testing"
	"time"
)

type forCaching struct {
	Caca string
}

func TestCache_Write_Read_Query(t *testing.T) {
	type args struct {
		q         interface{}
		variables map[string]interface{}
	}

	q1 := forCaching{"sasa"}

	tests := []struct {
		name     string
		bucket   string
		validity time.Duration
		args     args
		want     interface{}

		wantReadErr, wantWriteErr bool
	}{
		{
			"normal",
			"nrml",
			5 * time.Second,
			args{
				q: q1,
				variables: map[string]interface{}{
					"one":   "two",
					"three": 4,
				},
			},
			q1,
			false,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := cache.NewCache(tt.bucket, tt.validity)

			err = c.WriteQuery(tt.args.q, tt.args.variables)
			if (err != nil) != tt.wantWriteErr {
				t.Errorf("Cache.WriteQuery() error = %v, wantWriteErr %v", err, tt.wantWriteErr)
				return
			}

			got, err := c.ReadQuery(tt.args.q, tt.args.variables)
			if (err != nil) != tt.wantReadErr {
				t.Errorf("Cache.ReadQuery() error = %v, wantReadErr %v", err, tt.wantReadErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Cache.ReadQuery() = %v, want %v", got, tt.want)
			}
		})
	}
}
