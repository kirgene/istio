// Copyright 2018 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package spybackend

import (
	"fmt"
	"io/ioutil"
	"testing"

	"istio.io/api/mixer/adapter/model/v1beta1"
	istio_mixer_v1 "istio.io/api/mixer/v1"
	adapter_integration "istio.io/istio/mixer/pkg/adapter/test"
)

const (
	h1 = `
apiVersion: "config.istio.io/v1alpha2"
kind: handler
metadata:
  name: h1
  namespace: istio-system
spec:
  adapter: spybackend-nosession
  connection:
    address: "%s"
---
`
	i1Metric = `
apiVersion: "config.istio.io/v1alpha2"
kind: instance
metadata:
  name: i1metric
  namespace: istio-system
spec:
  template: metric
  params:
    value: request.size | 123
    dimensions:
      destination_service: "\"myservice\""
      response_code: "200"
---
`

	r1H1I1Metric = `
apiVersion: "config.istio.io/v1alpha2"
kind: rule
metadata:
  name: r1
  namespace: istio-system
spec:
  actions:
  - handler: h1.istio-system
    instances:
    - i1metric
---
`

	h2 = `
apiVersion: "config.istio.io/v1alpha2"
kind: handler
metadata:
  name: h2
  namespace: istio-system
spec:
  adapter: spybackend-nosession
  connection:
    address: "%s"
---
`

	i2Metric = `
apiVersion: "config.istio.io/v1alpha2"
kind: instance
metadata:
  name: i2metric
  namespace: istio-system
spec:
  template: metric
  params:
    value: request.size | 456
    dimensions:
      destination_service: "\"myservice2\""
      response_code: "400"
---
`

	r2H2I2Metric = `
apiVersion: "config.istio.io/v1alpha2"
kind: rule
metadata:
  name: r2
  namespace: istio-system
spec:
  actions:
  - handler: h2.istio-system
    instances:
    - i2metric
---
`
	i3List = `
apiVersion: "config.istio.io/v1alpha2"
kind: instance
metadata:
  name: i3list
  namespace: istio-system
spec:
  template: listentry
  params:
    value: source.name | "defaultstr"
---
`

	r3H1I3List = `
apiVersion: "config.istio.io/v1alpha2"
kind: rule
metadata:
  name: r3
  namespace: istio-system
spec:
  actions:
  - handler: h1.istio-system
    instances:
    - i3list
---
`

	i4Quota = `
apiVersion: "config.istio.io/v1alpha2"
kind: instance
metadata:
  name: requestQuota
  namespace: istio-system
spec:
  template: quota
  params:
    dimensions:
      source: source.labels["app"] | source.service | "unknown"
      sourceVersion: source.labels["version"] | "unknown"
      destination: destination.labels["app"] | destination.service | "unknown"
      destinationVersion: destination.labels["version"] | "unknown"
---
`

	r4h1i4Quota = `
apiVersion: "config.istio.io/v1alpha2"
kind: rule
metadata:
  name: r4
  namespace: istio-system
spec:
  actions:
  - handler: h1
    instances:
    - requestQuota
`

	r6MatchIfReqIDH1i4Metric = `
apiVersion: "config.istio.io/v1alpha2"
kind: rule
metadata:
  name: r5
  namespace: istio-system
spec:
  match: request.id | "unknown" != "unknown"
  actions:
  - handler: h1
    instances:
    - i1metric
`
)

func TestNoSessionBackend(t *testing.T) {

	testdata := []struct {
		name  string
		calls []adapter_integration.Call
		want  string
	}{
		{
			name: "single report call with attributes",
			calls: []adapter_integration.Call{
				{
					CallKind: adapter_integration.REPORT,
					Attrs:    map[string]interface{}{"request.size": int64(666)},
				},
			},
			want: `
		{
		 "AdapterState": [
		  {
		   "dedup_id": "stripped_for_test",
		   "instances": [
		    {
		     "dimensions": {
		      "destination_service": {
		       "Value": {
		        "StringValue": "myservice2"
		       }
		      },
		      "response_code": {
		       "Value": {
		        "Int64Value": 400
		       }
		      }
		     },
		     "name": "i2metric.instance.istio-system",
		     "value": {
		      "Value": {
		       "Int64Value": 666
		      }
		     }
		    }
		   ]
		  },
		  {
		   "dedup_id": "stripped_for_test",
		   "instances": [
		    {
		     "dimensions": {
		      "destination_service": {
		       "Value": {
		        "StringValue": "myservice"
		       }
		      },
		      "response_code": {
		       "Value": {
		        "Int64Value": 200
		       }
		      }
		     },
		     "name": "i1metric.instance.istio-system",
		     "value": {
		      "Value": {
		       "Int64Value": 666
		      }
		     }
		    }
		   ]
		  }
		 ],
		 "Returns": [
		  {
		   "Check": {
		    "Status": {},
		    "ValidDuration": 0,
		    "ValidUseCount": 0
		   },
		   "Quota": null,
		   "Error": null
		  }
		 ]
		}
`,
		},
		{
			name: "single report call no attributes",
			calls: []adapter_integration.Call{
				{
					CallKind: adapter_integration.REPORT,
					Attrs:    map[string]interface{}{},
				},
			},
			want: `
		{
		 "AdapterState": [
		  {
		   "dedup_id": "stripped_for_test",
		   "instances": [
		    {
		     "dimensions": {
		      "destination_service": {
		       "Value": {
		        "StringValue": "myservice2"
		       }
		      },
		      "response_code": {
		       "Value": {
		        "Int64Value": 400
		       }
		      }
		     },
		     "name": "i2metric.instance.istio-system",
		     "value": {
		      "Value": {
		       "Int64Value": 456
		      }
		     }
		    }
		   ]
		  },
		  {
		   "dedup_id": "stripped_for_test",
		   "instances": [
		    {
		     "dimensions": {
		      "destination_service": {
		       "Value": {
		        "StringValue": "myservice"
		       }
		      },
		      "response_code": {
		       "Value": {
		        "Int64Value": 200
		       }
		      }
		     },
		     "name": "i1metric.instance.istio-system",
		     "value": {
		      "Value": {
		       "Int64Value": 123
		      }
		     }
		    }
		   ]
		  }
		 ],
		 "Returns": [
		  {
		   "Check": {
		    "Status": {},
		    "ValidDuration": 0,
		    "ValidUseCount": 0
		   },
		   "Quota": null,
		   "Error": null
		  }
		 ]
		}
`,
		},
		{
			name: "single check call with attributes",
			calls: []adapter_integration.Call{
				{
					CallKind: adapter_integration.CHECK,
					Attrs:    map[string]interface{}{"source.name": "foobar"},
				},
			},
			want: `
   		{
    		 "AdapterState": [
    		  {
    		   "dedup_id": "stripped_for_test",
    		   "instance": {
    		    "name": "i3list.instance.istio-system",
    		    "value": "foobar"
    		   }
    		  }
    		 ],
    		 "Returns": [
    		  {
    		   "Check": {
    		    "Status": {},
    		    "ValidDuration": 0,
    		    "ValidUseCount": 31
    		   },
    		   "Quota": null,
    		   "Error": null
    		  }
    		 ]
    		}
`,
		},
		{
			name: "single check call no attributes",
			calls: []adapter_integration.Call{
				{
					CallKind: adapter_integration.CHECK,
					Attrs:    map[string]interface{}{},
				},
			},
			want: `
    		{
    		 "AdapterState": [
    		  {
    		   "dedup_id": "stripped_for_test",
    		   "instance": {
    		    "name": "i3list.instance.istio-system",
                "value": "defaultstr"
    		   }
    		  }
    		 ],
    		 "Returns": [
    		  {
    		   "Check": {
    		    "Status": {},
    		    "ValidDuration": 0,
    		    "ValidUseCount": 31
    		   },
    		   "Quota": null,
    		   "Error": null
    		  }
    		 ]
    		}
`,
		},
		{
			name: "single quota call with attributes",
			calls: []adapter_integration.Call{{
				CallKind: adapter_integration.CHECK,
				Quotas: map[string]istio_mixer_v1.CheckRequest_QuotaParams{
					"requestQuota": {
						Amount:     35,
						BestEffort: true,
					},
				},
				Attrs: map[string]interface{}{"source.service": "foobar"},
			}},
			want: `
    		{
    		 "AdapterState": [
    		  {
    		   "dedup_id": "stripped_for_test",
    		   "instance": {
    		    "name": "i3list.instance.istio-system",
    		    "value": "defaultstr"
    		   }
    		  },
    		  {
    		   "dedup_id": "stripped_for_test",
    		   "instance": {
    		    "dimensions": {
    		     "destination": {
    		      "Value": {
    		       "StringValue": "svc.cluster.local"
    		      }
    		     },
    		     "destinationVersion": {
    		      "Value": {
    		       "StringValue": "unknown"
    		      }
    		     },
    		     "source": {
    		      "Value": {
    		       "StringValue": "foobar"
    		      }
    		     },
    		     "sourceVersion": {
    		      "Value": {
    		       "StringValue": "unknown"
    		      }
    		     }
    		    },
    		    "name": "requestQuota.instance.istio-system"
    		   },
    		   "quota_request": {
    		    "quotas": {
    		     "requestQuota.instance.istio-system": {
    		      "amount": 35,
    		      "best_effort": true
    		     }
    		    }
    		   }
    		  }
    		 ],
    		 "Returns": [
    		  {
    		   "Check": {
    		    "Status": {},
    		    "ValidDuration": 0,
    		    "ValidUseCount": 0
    		   },
    		   "Quota": {
    		    "requestQuota": {
    		     "Status": {},
    		     "ValidDuration": 0,
    		     "Amount": 32
    		    }
    		   },
    		   "Error": null
    		  }
    		 ]
    		}
`,
		},
		{
			name: "single quota call no attributes",
			calls: []adapter_integration.Call{{
				CallKind: adapter_integration.CHECK,
				Quotas: map[string]istio_mixer_v1.CheckRequest_QuotaParams{
					"requestQuota": {
						Amount:     35,
						BestEffort: true,
					},
				},
			}},
			want: `
    		{
    		 "AdapterState": [
    		  {
    		   "dedup_id": "stripped_for_test",
    		   "instance": {
    		    "name": "i3list.instance.istio-system",
    		    "value": "defaultstr"
    		   }
    		  },
    		  {
    		   "dedup_id": "stripped_for_test",
    		   "instance": {
    		    "dimensions": {
    		     "destination": {
    		      "Value": {
    		       "StringValue": "svc.cluster.local"
    		      }
    		     },
    		     "destinationVersion": {
    		      "Value": {
    		       "StringValue": "unknown"
    		      }
    		     },
    		     "source": {
    		      "Value": {
    		       "StringValue": "unknown"
    		      }
    		     },
    		     "sourceVersion": {
    		      "Value": {
    		       "StringValue": "unknown"
    		      }
    		     }
    		    },
    		    "name": "requestQuota.instance.istio-system"
    		   },
    		   "quota_request": {
    		    "quotas": {
    		     "requestQuota.instance.istio-system": {
    		      "amount": 35,
    		      "best_effort": true
    		     }
    		    }
    		   }
    		  }
    		 ],
    		 "Returns": [
    		  {
    		   "Check": {
    		    "Status": {},
    		    "ValidDuration": 0,
    		    "ValidUseCount": 0
    		   },
    		   "Quota": {
    		    "requestQuota": {
    		     "Status": {},
    		     "ValidDuration": 0,
    		     "Amount": 32
    		    }
    		   },
    		   "Error": null
    		  }
    		 ]
    		}
`,
		},

		{
			name: "multiple mix calls",
			calls: []adapter_integration.Call{
				// 3 report calls; varying request.size attribute and no attributes call too.
				{
					CallKind: adapter_integration.REPORT,
					Attrs:    map[string]interface{}{"request.size": 666},
				},
				{
					CallKind: adapter_integration.REPORT,
					Attrs:    map[string]interface{}{"request.size": 888},
				},
				{
					CallKind: adapter_integration.REPORT,
				},

				// 3 check calls; varying source.name attribute and no attributes call too.,
				{
					CallKind: adapter_integration.CHECK,
					Attrs:    map[string]interface{}{"source.name": "foobar"},
				},
				{
					CallKind: adapter_integration.CHECK,
					Attrs:    map[string]interface{}{"source.name": "bazbaz"},
				},
				{
					CallKind: adapter_integration.CHECK,
				},

				// one call with quota args
				{
					CallKind: adapter_integration.CHECK,
					Quotas: map[string]istio_mixer_v1.CheckRequest_QuotaParams{
						"requestQuota": {
							Amount:     35,
							BestEffort: true,
						},
					},
				},
				// one report request with request.id to match r4 rule
				{
					CallKind: adapter_integration.REPORT,
					Attrs:    map[string]interface{}{"request.id": "somereqid"},
				},
			},

			// want:
			// * 4 i2metric.instance.istio-system for 4 report calls
			// * 5 i1metric.instance.istio-system for 4 report calls (3 report calls without request.id attribute and 1 report calls
			//     with request.id attribute, which result into 2 dispatch report rules to resolve successfully).
			// * 4 i3list.instance.istio-system for 4 check calls
			// * 1 requestQuota.instance.istio-system for 1 quota call
			want: `
    		{
    		 "AdapterState": [
    		  {
    		   "dedup_id": "stripped_for_test",
    		   "instances": [
    		    {
    		     "dimensions": {
    		      "destination_service": {
    		       "Value": {
    		        "StringValue": "myservice2"
    		       }
    		      },
    		      "response_code": {
    		       "Value": {
    		        "Int64Value": 400
    		       }
    		      }
    		     },
    		     "name": "i2metric.instance.istio-system",
    		     "value": {
    		      "Value": {
    		       "Int64Value": 456
    		      }
    		     }
    		    }
    		   ]
    		  },
    		  {
    		   "dedup_id": "stripped_for_test",
    		   "instances": [
    		    {
    		     "dimensions": {
    		      "destination_service": {
    		       "Value": {
    		        "StringValue": "myservice2"
    		       }
    		      },
    		      "response_code": {
    		       "Value": {
    		        "Int64Value": 400
    		       }
    		      }
    		     },
    		     "name": "i2metric.instance.istio-system",
    		     "value": {
    		      "Value": {
    		       "Int64Value": 456
    		      }
    		     }
    		    }
    		   ]
    		  },
    		  {
    		   "dedup_id": "stripped_for_test",
    		   "instances": [
    		    {
    		     "dimensions": {
    		      "destination_service": {
    		       "Value": {
    		        "StringValue": "myservice2"
    		       }
    		      },
    		      "response_code": {
    		       "Value": {
    		        "Int64Value": 400
    		       }
    		      }
    		     },
    		     "name": "i2metric.instance.istio-system",
    		     "value": {
    		      "Value": {
    		       "Int64Value": 456
    		      }
    		     }
    		    }
    		   ]
    		  },
    		  {
    		   "dedup_id": "stripped_for_test",
    		   "instances": [
    		    {
    		     "dimensions": {
    		      "destination_service": {
    		       "Value": {
    		        "StringValue": "myservice2"
    		       }
    		      },
    		      "response_code": {
    		       "Value": {
    		        "Int64Value": 400
    		       }
    		      }
    		     },
    		     "name": "i2metric.instance.istio-system",
    		     "value": {
    		      "Value": {
    		       "Int64Value": 456
    		      }
    		     }
    		    }
    		   ]
    		  },
    		  {
    		   "dedup_id": "stripped_for_test",
    		   "instances": [
    		    {
    		     "dimensions": {
    		      "destination_service": {
    		       "Value": {
    		        "StringValue": "myservice"
    		       }
    		      },
    		      "response_code": {
    		       "Value": {
    		        "Int64Value": 200
    		       }
    		      }
    		     },
    		     "name": "i1metric.instance.istio-system",
    		     "value": {
    		      "Value": {
    		       "Int64Value": 123
    		      }
    		     }
    		    }
    		   ]
    		  },
    		  {
    		   "dedup_id": "stripped_for_test",
    		   "instances": [
    		    {
    		     "dimensions": {
    		      "destination_service": {
    		       "Value": {
    		        "StringValue": "myservice"
    		       }
    		      },
    		      "response_code": {
    		       "Value": {
    		        "Int64Value": 200
    		       }
    		      }
    		     },
    		     "name": "i1metric.instance.istio-system",
    		     "value": {
    		      "Value": {
    		       "Int64Value": 123
    		      }
    		     }
    		    }
    		   ]
    		  },
    		  {
    		   "dedup_id": "stripped_for_test",
    		   "instances": [
    		    {
    		     "dimensions": {
    		      "destination_service": {
    		       "Value": {
    		        "StringValue": "myservice"
    		       }
    		      },
    		      "response_code": {
    		       "Value": {
    		        "Int64Value": 200
    		       }
    		      }
    		     },
    		     "name": "i1metric.instance.istio-system",
    		     "value": {
    		      "Value": {
    		       "Int64Value": 123
    		      }
    		     }
    		    }
    		   ]
    		  },
    		  {
    		   "dedup_id": "stripped_for_test",
    		   "instances": [
    		    {
    		     "dimensions": {
    		      "destination_service": {
    		       "Value": {
    		        "StringValue": "myservice"
    		       }
    		      },
    		      "response_code": {
    		       "Value": {
    		        "Int64Value": 200
    		       }
    		      }
    		     },
    		     "name": "i1metric.instance.istio-system",
    		     "value": {
    		      "Value": {
    		       "Int64Value": 123
    		      }
    		     }
    		    },
    		    {
    		     "dimensions": {
    		      "destination_service": {
    		       "Value": {
    		        "StringValue": "myservice"
    		       }
    		      },
    		      "response_code": {
    		       "Value": {
    		        "Int64Value": 200
    		       }
    		      }
    		     },
    		     "name": "i1metric.instance.istio-system",
    		     "value": {
    		      "Value": {
    		       "Int64Value": 123
    		      }
    		     }
    		    }
    		   ]
    		  },
    		  {
    		   "dedup_id": "stripped_for_test",
    		   "instance": {
    		    "name": "i3list.instance.istio-system",
    		    "value": "foobar"
    		   }
    		  },
    		  {
    		   "dedup_id": "stripped_for_test",
    		   "instance": {
    		    "name": "i3list.instance.istio-system",
    		    "value": "defaultstr"
    		   }
    		  },
    		  {
    		   "dedup_id": "stripped_for_test",
    		   "instance": {
    		    "name": "i3list.instance.istio-system",
    		    "value": "defaultstr"
    		   }
    		  },
    		  {
    		   "dedup_id": "stripped_for_test",
    		   "instance": {
    		    "name": "i3list.instance.istio-system",
    		    "value": "bazbaz"
    		   }
    		  },
    		  {
    		   "dedup_id": "stripped_for_test",
    		   "instance": {
    		    "dimensions": {
    		     "destination": {
    		      "Value": {
    		       "StringValue": "svc.cluster.local"
    		      }
    		     },
    		     "destinationVersion": {
    		      "Value": {
    		       "StringValue": "unknown"
    		      }
    		     },
    		     "source": {
    		      "Value": {
    		       "StringValue": "unknown"
    		      }
    		     },
    		     "sourceVersion": {
    		      "Value": {
    		       "StringValue": "unknown"
    		      }
    		     }
    		    },
    		    "name": "requestQuota.instance.istio-system"
    		   },
    		   "quota_request": {
    		    "quotas": {
    		     "requestQuota.instance.istio-system": {
    		      "amount": 35,
    		      "best_effort": true
    		     }
    		    }
    		   }
    		  }
    		 ],
    		 "Returns": [
    		  {
    		   "Check": {
    		    "Status": {},
    		    "ValidDuration": 0,
    		    "ValidUseCount": 0
    		   },
    		   "Quota": null,
    		   "Error": null
    		  },
    		  {
    		   "Check": {
    		    "Status": {},
    		    "ValidDuration": 0,
    		    "ValidUseCount": 0
    		   },
    		   "Quota": null,
    		   "Error": null
    		  },
    		  {
    		   "Check": {
    		    "Status": {},
    		    "ValidDuration": 0,
    		    "ValidUseCount": 0
    		   },
    		   "Quota": null,
    		   "Error": null
    		  },
    		  {
    		   "Check": {
    		    "Status": {},
    		    "ValidDuration": 0,
    		    "ValidUseCount": 31
    		   },
    		   "Quota": null,
    		   "Error": null
    		  },
    		  {
    		   "Check": {
    		    "Status": {},
    		    "ValidDuration": 0,
    		    "ValidUseCount": 31
    		   },
    		   "Quota": null,
    		   "Error": null
    		  },
    		  {
    		   "Check": {
    		    "Status": {},
    		    "ValidDuration": 0,
    		    "ValidUseCount": 31
    		   },
    		   "Quota": null,
    		   "Error": null
    		  },
    		  {
    		   "Check": {
    		    "Status": {},
    		    "ValidDuration": 0,
    		    "ValidUseCount": 0
    		   },
    		   "Quota": {
    		    "requestQuota": {
    		     "Status": {},
    		     "ValidDuration": 0,
    		     "Amount": 32
    		    }
    		   },
    		   "Error": null
    		  },
    		  {
    		   "Check": {
    		    "Status": {},
    		    "ValidDuration": 0,
    		    "ValidUseCount": 0
    		   },
    		   "Quota": null,
    		   "Error": null
    		  }
    		 ]
    		}
`,
		},
	}

	adptCfgBytes, err := ioutil.ReadFile("nosession.yaml")
	if err != nil {
		t.Fatalf("cannot open file: %v", err)
	}

	for _, td := range testdata {
		t.Run(td.name, func(tt *testing.T) {
			adapter_integration.RunTest(
				tt,
				nil,
				adapter_integration.Scenario{
					Setup: func() (interface{}, error) {
						args := DefaultArgs()
						args.Behavior.HandleMetricResult = &v1beta1.ReportResult{}
						args.Behavior.HandleListEntryResult = &v1beta1.CheckResult{ValidUseCount: 31}
						args.Behavior.HandleQuotaResult = &v1beta1.QuotaResult{
							Quotas: map[string]v1beta1.QuotaResult_Result{"requestQuota.instance.istio-system": {GrantedAmount: 32}}}

						var s Server
						var err error
						if s, err = NewNoSessionServer(args); err != nil {
							return nil, err
						}
						s.Run()
						return s, nil
					},
					Teardown: func(ctx interface{}) {
						_ = ctx.(Server).Close()
					},
					GetState: func(ctx interface{}) (interface{}, error) {
						s := ctx.(*NoSessionServer)
						return s.GetState(), nil
					},
					SingleThreaded: false,
					ParallelCalls:  td.calls,
					GetConfig: func(ctx interface{}) ([]string, error) {
						s := ctx.(Server)
						return []string{
							// CRs for built-in templates are automatically added by the integration test framework.
							string(adptCfgBytes),
							fmt.Sprintf(h1, s.Addr().String()),
							i1Metric,
							r1H1I1Metric,
							fmt.Sprintf(h2, s.Addr().String()),
							i2Metric,
							r2H2I2Metric,
							i3List,
							r3H1I3List,
							i4Quota,
							r4h1i4Quota,
							r6MatchIfReqIDH1i4Metric,
						}, nil
					},
					Want: td.want,
				},
			)
		})
	}
}
