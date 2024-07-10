package certainly

import (
	"context"

	"github.com/mholt/acmez/v2/acme"
)

// ChallengeProvider implements go-acme/lego Provider interface which is used for ACME DNS challenge handling
type ChallengeProvider struct {
	servers []CertainlyNS
}

// NewChallengeProvider creates a new instance of ChallengeProvider
func NewChallengeProvider(servers []CertainlyNS) ChallengeProvider {
	return ChallengeProvider{servers: servers}
}

// Present is used for making the ACME DNS challenge token available for DNS
func (c *ChallengeProvider) Present(ctx context.Context, challenge acme.Challenge) error {
	for _, s := range c.servers {
		s.SetChallengeToken(challenge.DNS01TXTRecordName(), challenge.DNS01KeyAuthorization())
	}
	return nil
}

// CleanUp is called after the run to remove the ACME DNS challenge tokens from DNS records
func (c *ChallengeProvider) CleanUp(ctx context.Context, _ acme.Challenge) error {
	return nil
}

// Wait is a dummy function as we are just going to be ready to answer the challenge from the get-go
func (c *ChallengeProvider) Wait(_ context.Context, _ acme.Challenge) error {
	return nil
}
