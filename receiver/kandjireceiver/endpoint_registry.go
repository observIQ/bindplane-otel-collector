package kandjireceiver

var EndpointRegistry = map[KandjiEndpoint]EndpointSpec{}

func init() {
	maxLimit := 500

	EndpointRegistry[EPAuditEventsList] = EndpointSpec{
		Method:             "GET",
		Path:               "/audit/events",
		Description:        "List audit log events.",
		SupportsPagination: true,
		Params: []ParamSpec{
			{
				Name: "limit",
				Type: ParamInt,
				Constraints: &ParamConstraints{
					MaxInt: &maxLimit,
				},
			},
			{
				Name:        "sort_by",
				Type:        ParamString,
				AllowedVals: []string{"occurred_at", "-occurred_at"},
			},
			CursorParam,
		},
		ResponseType: AuditEventsResponse{},
	}
}
