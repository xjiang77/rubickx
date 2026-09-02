package facade

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }

type artifactValidator struct{}

func (artifactValidator) validate(artifact string) (string, error) {
	if artifact == "" {
		return "", &PatternError{code: "invalid_artifact"}
	}
	return "validate:" + artifact, nil
}

type deployer struct{}

func (deployer) deploy(artifact string) string { return "deploy:" + artifact }

type healthVerifier struct{}

func (healthVerifier) verify(artifact string, healthy bool) (string, error) {
	if !healthy {
		return "", &PatternError{code: "health_check_failed"}
	}
	return "verify:" + artifact, nil
}

type releaseFacade struct {
	validator artifactValidator
	deployer  deployer
	verifier  healthVerifier
}

func (f releaseFacade) release(request map[string]any) (map[string]any, error) {
	artifact, _ := request["artifact"].(string)
	first, err := f.validator.validate(artifact)
	if err != nil {
		return nil, err
	}
	steps := []string{first}
	if dry, _ := request["dry_run"].(bool); dry {
		steps = append(steps, "plan:"+artifact)
		return map[string]any{"artifact": artifact, "status": "planned", "steps": steps}, nil
	}
	steps = append(steps, f.deployer.deploy(artifact))
	verified, err := f.verifier.verify(artifact, request["healthy"] == true)
	if err != nil {
		return nil, err
	}
	steps = append(steps, verified)
	return map[string]any{"artifact": artifact, "status": "released", "steps": steps}, nil
}
func Evaluate(input map[string]any) (any, error) {
	facade := releaseFacade{}
	raw, _ := input["releases"].([]any)
	releases := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		result, err := facade.release(item.(map[string]any))
		if err != nil {
			return nil, err
		}
		releases = append(releases, result)
	}
	return map[string]any{"releases": releases}, nil
}
