package server

func DefaultCatalog() Catalog {
	return Catalog{
		Algorithms: []CatalogItem{
			{ID: AlgorithmFixedWindow, Label: "Fixed Window", Description: "One counter per fixed time window."},
			{ID: AlgorithmSlidingWindowLog, Label: "Sliding Window Log", Description: "Exact timestamps inside the rolling window."},
			{ID: AlgorithmSlidingWindowCount, Label: "Sliding Window Counter", Description: "Weighted current and previous fixed windows."},
			{ID: AlgorithmTokenBucket, Label: "Token Bucket", Description: "Burst capacity with continuous refill."},
			{ID: AlgorithmLeakyBucket, Label: "Leaky Bucket", Description: "Bounded queue drained at a stable rate."},
		},
		Languages: []CatalogItem{
			{ID: LanguagePython, Label: "Python"},
			{ID: LanguageGo, Label: "Go", Debuggable: true},
			{ID: LanguageJava, Label: "Java"},
			{ID: LanguageJavaScript, Label: "JavaScript"},
		},
		Scenarios: defaultScenarios(),
		Modes:     []string{"semantic", "debug"},
	}
}

func defaultScenarios() []CatalogItem {
	windowConfig := map[string]float64{"limit": 3, "windowMs": 1000}
	rateConfig := map[string]float64{"capacity": 3, "ratePerSecond": 2}
	scenario := func(id, label, tier, algorithm, lesson string, config map[string]float64, brief *ScenarioBrief, at ...int64) CatalogItem {
		timeline := make([]RequestPoint, 0, len(at))
		for _, timestamp := range at {
			timeline = append(timeline, RequestPoint{AtMs: timestamp, Cost: 1, Key: "alice"})
		}
		return CatalogItem{ID: id, Label: label, Tier: tier, Algorithm: algorithm, DefaultLanguage: LanguageGo, DefaultConfig: config, RequestTimeline: timeline, Lesson: lesson, Brief: brief}
	}
	concurrent := scenario(
		"concurrent-callers",
		"Concurrent callers",
		"core",
		AlgorithmSlidingWindowCount,
		"JavaScript synchronous calls stay in one event-loop turn, but read-modify-write across await can race; shared quotas still need Redis atomicity.",
		windowConfig,
		coreScenarioBrief(
			"alice + bob · 3 requests each at 0 / 100 / 200 ms",
			"ALLOW ×6; alice and bob each use an independent limit 3.",
			"allow", "allow", "allow", "allow", "allow", "allow",
		),
		0, 0, 100, 100, 200, 200,
	)
	concurrent.RequestTimeline[1].Key = "bob"
	concurrent.RequestTimeline[3].Key = "bob"
	concurrent.RequestTimeline[5].Key = "bob"
	return []CatalogItem{
		scenario(
			"steady-traffic", "Steady traffic", "core", AlgorithmTokenBucket,
			"A sustainable arrival rate keeps every request allowed.", rateConfig,
			coreScenarioBrief(
				"alice · 4 requests at 0 / 500 / 1,000 / 1,500 ms",
				"ALLOW ×4; refill keeps pace with every request.",
				"allow", "allow", "allow", "allow",
			),
			0, 500, 1000, 1500,
		),
		scenario(
			"burst-capacity", "Burst capacity", "core", AlgorithmTokenBucket,
			"A token bucket admits a short burst, then exposes refill timing.", rateConfig,
			coreScenarioBrief(
				"alice · 4× at 0 ms, then 1× at 500 ms",
				"ALLOW ×3 → DENY ×1; after 500 ms of refill, request #5 is ALLOW.",
				"allow", "allow", "allow", "deny", "allow",
			),
			0, 0, 0, 0, 500,
		),
		scenario(
			"window-boundary", "Window boundary", "core", AlgorithmFixedWindow,
			"Fixed windows can admit two bursts around a boundary.", windowConfig,
			coreScenarioBrief(
				"alice · 6 requests at 800 / 900 / 999 / 1,000 / 1,001 / 1,100 ms",
				"ALLOW ×6; [0, 1,000) and [1,000, 2,000) each admit three requests.",
				"allow", "allow", "allow", "allow", "allow", "allow",
			),
			800, 900, 999, 1000, 1001, 1100,
		),
		scenario(
			"exactness-vs-memory", "Exactness vs memory", "core", AlgorithmSlidingWindowLog,
			"The log is exact because it retains every request timestamp.", windowConfig,
			coreScenarioBrief(
				"alice · 5 requests at 0 / 100 / 200 / 900 / 1,101 ms",
				"ALLOW ×3 → DENY (retry 100 ms) → ALLOW at 1,101 ms after eviction.",
				"allow", "allow", "allow", "deny", "allow",
			),
			0, 100, 200, 900, 1101,
		),
		scenario(
			"smoothed-output", "Smoothed output", "core", AlgorithmLeakyBucket,
			"A leaky bucket turns bursty input into bounded queued work.", rateConfig,
			coreScenarioBrief(
				"alice · 4× at 0 ms, then 1× at 500 / 1,000 ms",
				"ALLOW ×3 → DENY → ALLOW at 500 and 1,000 ms as the queue drains.",
				"allow", "allow", "allow", "deny", "allow", "allow",
			),
			0, 0, 0, 0, 500, 1000,
		),
		concurrent,
		scenario(
			"clock-input-safety", "Clock and input safety", "core", AlgorithmTokenBucket,
			"Injected monotonic time makes refill behavior deterministic and testable.", rateConfig,
			coreScenarioBrief(
				"alice · 4 requests at 0 / 250 / 500 / 1,000 ms",
				"ALLOW ×4; remaining is 2 / 1.5 / 1 / 1 after each request.",
				"allow", "allow", "allow", "allow",
			),
			0, 250, 500, 1000,
		),
		scenario(
			"policy-composition", "Policy composition", "system", AlgorithmFixedWindow,
			"Compose per-user and per-endpoint policies without hiding which policy rejected.", windowConfig,
			systemScenarioBrief(
				"per-client 3 · endpoint-wide 6 · window 1,000 ms",
				"One client reaches its quota; multiple clients share the endpoint quota.",
				"The response identifies every policy that rejected the request.",
				expectedCase("Same client · requests #1–3", "200 · all policies allowed", "allow"),
				expectedCase("Same client · request #4", "429 · rejected by per-client", "deny"),
				expectedCase("Endpoint-wide request #7", "429 · rejected by endpoint-wide", "deny"),
			),
			0, 100, 200, 300,
		),
		scenario(
			"http-contract", "HTTP 429 contract", "system", AlgorithmFixedWindow,
			"Clients need remaining, reset and retry headers, not only status 429.", windowConfig,
			systemScenarioBrief(
				"",
				"alice · 4 requests inside one window",
				"A rejected request carries the headers a client needs to back off.",
				expectedCase("Requests #1–3", "200 · ALLOW", "allow"),
				expectedCase("Request #4", "429 · RateLimit-* + Retry-After", "deny"),
			),
			0, 0, 0, 0,
		),
		scenario(
			"local-vs-shared", "Local vs shared quota", "system", AlgorithmFixedWindow,
			"An in-memory limiter is process-local; Redis shares quota across replicas.", windowConfig,
			replicaScopedBrief(systemScenarioBrief(
				"",
				"alice · requests alternate between Replica A and Replica B",
				"The same client sees different quota scope when the backing store changes.",
				expectedCase("Memory · Replica A and B", "Each replica has its own limit 3", "observe"),
				expectedCase("Redis healthy · Replica A and B", "Both replicas share one limit 3", "observe"),
			)),
			0, 0, 100, 100,
		),
		scenario(
			"redis-atomicity", "Redis atomicity", "system", AlgorithmFixedWindow,
			"Lua keeps increment, expiry and TTL observation in one atomic operation.", windowConfig,
			systemScenarioBrief(
				"",
				"alice · Burst ×10",
				"Redis health and the failure policy determine whether the shared quota is enforced.",
				expectedCase("Redis healthy", "3×200 + 7×429 · atomic increment + TTL", "observe"),
				expectedCase("Redis unavailable · fail-open", "200 + degraded + bypass", "allow"),
				expectedCase("Redis unavailable · fail-closed", "503 + degraded + enforced", "deny"),
			),
			0, 0, 0, 0,
		),
		scenario(
			"redis-outage", "Redis outage", "system", AlgorithmFixedWindow,
			"Fail-open protects availability; fail-closed protects quota enforcement.", windowConfig,
			systemScenarioBrief(
				"",
				"alice · requests while Redis is unavailable",
				"The selected failure policy explicitly trades availability for enforcement.",
				expectedCase("Redis healthy", "Quota is enforced normally", "observe"),
				expectedCase("Redis unavailable · fail-open", "200 + degraded + bypass", "allow"),
				expectedCase("Redis unavailable · fail-closed", "503 + degraded + enforced", "deny"),
			),
			0, 100, 200,
		),
		scenario(
			"hot-key-sharding", "Hot key and sharding", "system", AlgorithmFixedWindow,
			"A single popular quota key can become the storage bottleneck.", windowConfig,
			systemScenarioBrief(
				"",
				"One repeated hot key compared with distinct client keys",
				"Sharding distributes keys, but one quota key still shares one limit and one shard.",
				expectedCase("Redis healthy · same key", "Stable shard · first 3 requests return 200, then 429", "observe"),
				expectedCase("Redis healthy · distinct keys", "Keys can spread across four shards", "observe"),
				expectedCase("Redis unavailable · fail-open", "200 + degraded + bypass", "allow"),
				expectedCase("Redis unavailable · fail-closed", "503 + degraded + enforced", "deny"),
			),
			0, 0, 0, 0,
		),
		scenario(
			"multi-region-quota", "Multi-region quota", "system", AlgorithmFixedWindow,
			"Global accuracy trades latency and availability against regional allocation.", windowConfig,
			conceptualBrief(systemScenarioBrief(
				"No live limiter",
				"Conceptual quota allocation across regions; no HTTP requests are sent",
				"There is no single best design across latency, availability, and global accuracy.",
				expectedCase("Region-local quota", "Low latency, weaker global accuracy", "observe"),
				expectedCase("Globally coordinated quota", "Higher accuracy, added latency and dependency", "observe"),
				expectedCase("Regional allocation", "Bounded overshoot with explicit rebalancing", "observe"),
			)),
			0, 100, 200, 300,
		),
	}
}

func coreScenarioBrief(traffic, summary string, admissions ...string) *ScenarioBrief {
	return &ScenarioBrief{
		Traffic: traffic,
		Expected: ScenarioExpectation{
			Summary:    summary,
			Admissions: admissions,
		},
	}
}

func systemScenarioBrief(policy, traffic, summary string, cases ...ScenarioExpectedCase) *ScenarioBrief {
	return &ScenarioBrief{
		Policy:  policy,
		Traffic: traffic,
		Expected: ScenarioExpectation{
			Summary: summary,
			Cases:   cases,
		},
	}
}

func replicaScopedBrief(brief *ScenarioBrief) *ScenarioBrief {
	brief.ReplicaScoped = true
	return brief
}

func conceptualBrief(brief *ScenarioBrief) *ScenarioBrief {
	brief.Conceptual = true
	return brief
}

func expectedCase(when, result, kind string) ScenarioExpectedCase {
	return ScenarioExpectedCase{When: when, Result: result, Kind: kind}
}
