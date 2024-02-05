package main

import (
	"context"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	nanomdm_mock "github.com/fleetdm/fleet/v4/server/mock/nanomdm"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/micromdm/nanodep/tokenpki"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestBasicGitOps(t *testing.T) {
	_, ds := runServerWithMockedDS(t)

	ds.BatchSetScriptsFunc = func(ctx context.Context, tmID *uint, scripts []*fleet.Script) error { return nil }
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		return nil
	}
	ds.ListGlobalPoliciesFunc = func(ctx context.Context, opts fleet.ListOptions) ([]*fleet.Policy, error) { return nil, nil }
	ds.ListQueriesFunc = func(ctx context.Context, opts fleet.ListQueryOptions) ([]*fleet.Query, error) { return nil, nil }

	// Mock appConfig
	savedAppConfig := &fleet.AppConfig{}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.SaveAppConfigFunc = func(ctx context.Context, config *fleet.AppConfig) error {
		savedAppConfig = config
		return nil
	}
	var enrolledSecrets []*fleet.EnrollSecret
	ds.ApplyEnrollSecretsFunc = func(ctx context.Context, teamID *uint, secrets []*fleet.EnrollSecret) error {
		enrolledSecrets = secrets
		return nil
	}

	tmpFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)

	const (
		fleetServerURL = "https://fleet.example.com"
		orgName        = "GitOps Test"
	)
	t.Setenv("FLEET_SERVER_URL", fleetServerURL)

	_, err = tmpFile.WriteString(
		`
controls:
queries:
policies:
agent_options:
org_settings:
  server_settings:
    server_url: $FLEET_SERVER_URL
  org_info:
    contact_url: https://example.com/contact
    org_logo_url: ""
    org_logo_url_light_background: ""
    org_name: ${ORG_NAME}
  secrets:
`,
	)
	require.NoError(t, err)

	// No file
	var errWriter strings.Builder
	_, err = runAppNoChecks([]string{"gitops", tmpFile.Name()})
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "-f must be specified")

	// Bad file
	errWriter.Reset()
	_, err = runAppNoChecks([]string{"gitops", "-f", "fileDoesNotExist.yml"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")

	// Empty file
	errWriter.Reset()
	badFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = runAppNoChecks([]string{"gitops", "-f", badFile.Name()})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "YAML processing errors")

	// DoGitOps error
	t.Setenv("ORG_NAME", "")
	_, err = runAppNoChecks([]string{"gitops", "-f", tmpFile.Name()})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "organization name must be present")

	// Dry run
	t.Setenv("ORG_NAME", orgName)
	_ = runAppForTest(t, []string{"gitops", "-f", tmpFile.Name(), "--dry-run"})
	assert.Equal(t, fleet.AppConfig{}, *savedAppConfig, "AppConfig should be empty")

	// Real run
	_ = runAppForTest(t, []string{"gitops", "-f", tmpFile.Name()})
	assert.Equal(t, orgName, savedAppConfig.OrgInfo.OrgName)
	assert.Equal(t, fleetServerURL, savedAppConfig.ServerSettings.ServerURL)
	assert.Empty(t, enrolledSecrets)
}

func TestFullGitOps(t *testing.T) {
	license := &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}

	// mdm test configuration must be set so that activating windows MDM works.
	testCert, testKey, err := apple_mdm.NewSCEPCACertKey()
	require.NoError(t, err)
	testCertPEM := tokenpki.PEMCertificate(testCert.Raw)
	testKeyPEM := tokenpki.PEMRSAPrivateKey(testKey)
	fleetCfg := config.TestConfig()
	config.SetTestMDMConfig(t, &fleetCfg, testCertPEM, testKeyPEM, nil, "../../server/service/testdata")

	_, ds := runServerWithMockedDS(
		t, &service.TestServerOpts{
			License:     license,
			MDMStorage:  new(nanomdm_mock.Storage),
			MDMPusher:   mockPusher{},
			FleetConfig: &fleetCfg,
		},
	)

	var appliedScripts []*fleet.Script
	ds.BatchSetScriptsFunc = func(ctx context.Context, tmID *uint, scripts []*fleet.Script) error {
		appliedScripts = scripts
		return nil
	}
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		return nil
	}
	var appliedMacProfiles []*fleet.MDMAppleConfigProfile
	var appliedWinProfiles []*fleet.MDMWindowsConfigProfile
	ds.BatchSetMDMProfilesFunc = func(
		ctx context.Context, tmID *uint, macProfiles []*fleet.MDMAppleConfigProfile, winProfiles []*fleet.MDMWindowsConfigProfile,
	) error {
		appliedMacProfiles = macProfiles
		appliedWinProfiles = winProfiles
		return nil
	}
	ds.BulkSetPendingMDMHostProfilesFunc = func(ctx context.Context, hostIDs, teamIDs []uint, profileUUIDs, hostUUIDs []string) error {
		return nil
	}
	ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
		return job, nil
	}

	// Policies
	policy := fleet.Policy{}
	policy.ID = 1
	policy.Name = "Policy to delete"
	policyDeleted := false
	ds.ListGlobalPoliciesFunc = func(ctx context.Context, opts fleet.ListOptions) ([]*fleet.Policy, error) {
		return []*fleet.Policy{&policy}, nil
	}
	ds.PoliciesByIDFunc = func(ctx context.Context, ids []uint) (map[uint]*fleet.Policy, error) {
		if slices.Contains(ids, 1) {
			return map[uint]*fleet.Policy{1: &policy}, nil
		}
		return nil, nil
	}
	ds.DeleteGlobalPoliciesFunc = func(ctx context.Context, ids []uint) ([]uint, error) {
		policyDeleted = true
		assert.Equal(t, []uint{policy.ID}, ids)
		return ids, nil
	}
	var appliedPolicySpecs []*fleet.PolicySpec
	ds.ApplyPolicySpecsFunc = func(ctx context.Context, authorID uint, specs []*fleet.PolicySpec) error {
		appliedPolicySpecs = specs
		return nil
	}

	// Queries
	query := fleet.Query{}
	query.ID = 1
	query.Name = "Query to delete"
	queryDeleted := false
	ds.ListQueriesFunc = func(ctx context.Context, opts fleet.ListQueryOptions) ([]*fleet.Query, error) {
		return []*fleet.Query{&query}, nil
	}
	ds.DeleteQueriesFunc = func(ctx context.Context, ids []uint) (uint, error) {
		queryDeleted = true
		assert.Equal(t, []uint{query.ID}, ids)
		return 1, nil
	}
	ds.QueryFunc = func(ctx context.Context, id uint) (*fleet.Query, error) {
		if id == query.ID {
			return &query, nil
		}
		return nil, nil
	}
	var appliedQueries []*fleet.Query
	ds.QueryByNameFunc = func(ctx context.Context, teamID *uint, name string) (*fleet.Query, error) {
		return nil, &notFoundError{}
	}
	ds.ApplyQueriesFunc = func(
		ctx context.Context, authorID uint, queries []*fleet.Query, queriesToDiscardResults map[uint]struct{},
	) error {
		appliedQueries = queries
		return nil
	}

	// Mock appConfig
	savedAppConfig := &fleet.AppConfig{}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: true}}, nil
	}
	ds.SaveAppConfigFunc = func(ctx context.Context, config *fleet.AppConfig) error {
		savedAppConfig = config
		return nil
	}
	var enrolledSecrets []*fleet.EnrollSecret
	ds.ApplyEnrollSecretsFunc = func(ctx context.Context, teamID *uint, secrets []*fleet.EnrollSecret) error {
		enrolledSecrets = secrets
		return nil
	}

	const (
		fleetServerURL = "https://fleet.example.com"
		orgName        = "GitOps Test"
	)
	t.Setenv("FLEET_SERVER_URL", fleetServerURL)
	t.Setenv("ORG_NAME", orgName)

	// Dry run
	file := "./testdata/gitops/global_config_no_paths.yml"
	_ = runAppForTest(t, []string{"gitops", "-f", file, "--dry-run"})
	assert.Equal(t, fleet.AppConfig{}, *savedAppConfig, "AppConfig should be empty")
	assert.Len(t, enrolledSecrets, 0)
	assert.Len(t, appliedPolicySpecs, 0)
	assert.Len(t, appliedQueries, 0)
	assert.Len(t, appliedScripts, 0)
	assert.Len(t, appliedMacProfiles, 0)
	assert.Len(t, appliedWinProfiles, 0)

	// Real run
	_ = runAppForTest(t, []string{"gitops", "-f", file})
	assert.Equal(t, orgName, savedAppConfig.OrgInfo.OrgName)
	assert.Equal(t, fleetServerURL, savedAppConfig.ServerSettings.ServerURL)
	assert.Contains(t, string(*savedAppConfig.AgentOptions), "distributed_denylist_duration")
	assert.Len(t, enrolledSecrets, 2)
	assert.True(t, policyDeleted)
	assert.Len(t, appliedPolicySpecs, 5)
	assert.True(t, queryDeleted)
	assert.Len(t, appliedQueries, 3)
	assert.Len(t, appliedScripts, 1)
	assert.Len(t, appliedMacProfiles, 1)
	assert.Len(t, appliedWinProfiles, 1)
}
