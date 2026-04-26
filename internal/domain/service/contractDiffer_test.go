package service_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/service"
	diffvo "github.com/jitctx/jitctx/internal/domain/vo/diff"
)

// newDiffer is a helper that wires a ContractDiffer with a real SignatureNormalizer.
func newDiffer() service.ContractDiffer {
	return service.NewContractDiffer(service.NewSignatureNormalizer())
}

// specContract builds a minimal SpecContract with only the fields the tests care about.
func specContract(name string, contractType model.ContractType, methods ...string) model.SpecContract {
	return model.SpecContract{
		Name:    name,
		Type:    contractType,
		Methods: methods,
	}
}

// manifestContract builds a minimal manifest Contract.
func manifestContract(name string, contractType model.ContractType, signatures ...string) model.Contract {
	methods := make([]model.Method, 0, len(signatures))
	for _, sig := range signatures {
		methods = append(methods, model.Method{Signature: sig})
	}
	return model.Contract{
		Name:    name,
		Types:   []string{string(contractType)},
		Methods: methods,
	}
}

// findAction searches actions for one with the given contract name and returns it.
func findAction(actions []diffvo.DiffAction, name string) (diffvo.DiffAction, bool) {
	for _, a := range actions {
		if a.ContractName == name {
			return a, true
		}
	}
	return diffvo.DiffAction{}, false
}

// ─── CREATE ──────────────────────────────────────────────────────────────────

func TestContractDiffer_Diff_SpecOnlyProducesCREATE(t *testing.T) {
	t.Parallel()

	d := newDiffer()

	spec := []model.SpecContract{
		specContract("ChangeUserStatusUseCase", model.ContractInputPort, "void execute(ChangeUserStatusCommand cmd)"),
	}
	manifest := []model.Contract{}

	actions := d.Diff(spec, manifest)

	require.Len(t, actions, 1)
	a := actions[0]
	require.Equal(t, diffvo.DiffActionCreate, a.Type)
	require.Equal(t, "ChangeUserStatusUseCase", a.ContractName)
	require.Equal(t, string(model.ContractInputPort), a.ContractType)
	require.Equal(t, diffvo.DiffSeverityError, a.Severity)
	require.Empty(t, a.AddedMethods)
	require.Empty(t, a.RemovedMethods)
}

// ─── MODIFY ───────────────────────────────────────────────────────────────────

func TestContractDiffer_Diff_MethodDeltaProducesMODIFY(t *testing.T) {
	t.Parallel()

	d := newDiffer()

	spec := []model.SpecContract{
		specContract("UserRepository", model.ContractOutputPort,
			"User save(User user)",
			"Optional<User> findById(UUID id)",
		),
	}
	manifest := []model.Contract{
		manifestContract("UserRepository", model.ContractOutputPort,
			"Optional<User> findById(UUID id)",
			"void persist(User user)",
		),
	}

	actions := d.Diff(spec, manifest)

	require.Len(t, actions, 1)
	a := actions[0]
	require.Equal(t, diffvo.DiffActionModify, a.Type)
	require.Equal(t, "UserRepository", a.ContractName)
	require.Equal(t, diffvo.DiffSeverityWarning, a.Severity)
	// "save" is in spec but not in manifest → added
	require.Equal(t, []string{"save"}, a.AddedMethods)
	// "persist" is in manifest but not in spec → removed
	require.Equal(t, []string{"persist"}, a.RemovedMethods)
}

func TestContractDiffer_Diff_MODIFYListsAreSorted(t *testing.T) {
	t.Parallel()

	d := newDiffer()

	spec := []model.SpecContract{
		specContract("OrderService", model.ContractService,
			"void cancel(Order order)",
			"void approve(Order order)",
		),
	}
	manifest := []model.Contract{
		manifestContract("OrderService", model.ContractService,
			"void reject(Order order)",
			"void archive(Order order)",
		),
	}

	actions := d.Diff(spec, manifest)

	require.Len(t, actions, 1)
	a := actions[0]
	require.Equal(t, diffvo.DiffActionModify, a.Type)
	// Both lists must be alphabetically sorted.
	require.Equal(t, []string{"approve", "cancel"}, a.AddedMethods)
	require.Equal(t, []string{"archive", "reject"}, a.RemovedMethods)
}

// ─── NO-OP (skip) ─────────────────────────────────────────────────────────────

func TestContractDiffer_Diff_IdenticalSignaturesProducesNoAction(t *testing.T) {
	t.Parallel()

	d := newDiffer()

	spec := []model.SpecContract{
		specContract("UserRepository", model.ContractOutputPort,
			"User save(User user)",
			"Optional<User> findById(UUID id)",
		),
	}
	manifest := []model.Contract{
		manifestContract("UserRepository", model.ContractOutputPort,
			"User save(User user)",
			"Optional<User> findById(UUID id)",
		),
	}

	actions := d.Diff(spec, manifest)

	require.Empty(t, actions)
}

// ─── EXTRA ────────────────────────────────────────────────────────────────────

func TestContractDiffer_Diff_ManifestOnlyProducesEXTRA(t *testing.T) {
	t.Parallel()

	d := newDiffer()

	spec := []model.SpecContract{}
	manifest := []model.Contract{
		manifestContract("DeprecatedHelper", model.ContractService, "void doStuff()"),
	}

	actions := d.Diff(spec, manifest)

	require.Len(t, actions, 1)
	a := actions[0]
	require.Equal(t, diffvo.DiffActionExtra, a.Type)
	require.Equal(t, "DeprecatedHelper", a.ContractName)
	require.Equal(t, []string{string(model.ContractService)}, a.ContractTypes)
	require.Equal(t, diffvo.DiffSeverityInfo, a.Severity)
	require.Equal(t, -1, a.Layer)
}

// ─── WHITESPACE-INSENSITIVE MATCH (Gherkin scenario 6) ───────────────────────

func TestContractDiffer_Diff_WhitespaceInsensitiveSignatureProducesNoAction(t *testing.T) {
	t.Parallel()

	d := newDiffer()

	// Spec has extra spaces inside parens; manifest has compact form.
	spec := []model.SpecContract{
		specContract("UserRepository", model.ContractOutputPort,
			"User save( User user )",
		),
	}
	manifest := []model.Contract{
		manifestContract("UserRepository", model.ContractOutputPort,
			"User save(User user)",
		),
	}

	actions := d.Diff(spec, manifest)

	// Whitespace difference should NOT produce a MODIFY.
	require.Empty(t, actions)
}

// ─── EMPTY SPEC ───────────────────────────────────────────────────────────────

func TestContractDiffer_Diff_EmptySpecAllManifestContractsAreEXTRA(t *testing.T) {
	t.Parallel()

	d := newDiffer()

	spec := []model.SpecContract{}
	manifest := []model.Contract{
		manifestContract("UserRepository", model.ContractOutputPort, "User save(User user)"),
		manifestContract("UserService", model.ContractService, "void activate(UUID id)"),
	}

	actions := d.Diff(spec, manifest)

	require.Len(t, actions, 2)
	for _, a := range actions {
		require.Equal(t, diffvo.DiffActionExtra, a.Type)
		require.Equal(t, diffvo.DiffSeverityInfo, a.Severity)
		require.Equal(t, -1, a.Layer)
	}

	// Both manifest contracts must appear.
	names := make([]string, 0, len(actions))
	for _, a := range actions {
		names = append(names, a.ContractName)
	}
	require.ElementsMatch(t, []string{"UserRepository", "UserService"}, names)
}

// ─── EMPTY MANIFEST ───────────────────────────────────────────────────────────

func TestContractDiffer_Diff_EmptyManifestAllSpecContractsAreCREATE(t *testing.T) {
	t.Parallel()

	d := newDiffer()

	spec := []model.SpecContract{
		specContract("UserRepository", model.ContractOutputPort, "User save(User user)"),
		specContract("UserService", model.ContractService, "void activate(UUID id)"),
	}
	manifest := []model.Contract{}

	actions := d.Diff(spec, manifest)

	require.Len(t, actions, 2)
	for _, a := range actions {
		require.Equal(t, diffvo.DiffActionCreate, a.Type)
		require.Equal(t, diffvo.DiffSeverityError, a.Severity)
	}

	names := make([]string, 0, len(actions))
	for _, a := range actions {
		names = append(names, a.ContractName)
	}
	require.ElementsMatch(t, []string{"UserRepository", "UserService"}, names)
}

// ─── MULTIPLE METHODS, PARTIAL OVERLAP ────────────────────────────────────────

func TestContractDiffer_Diff_PartialMethodOverlapSetsAddedAndRemoved(t *testing.T) {
	t.Parallel()

	d := newDiffer()

	spec := []model.SpecContract{
		specContract("UserRepository", model.ContractOutputPort,
			"User save(User user)",
			"Optional<User> findByEmail(String email)",
			"void delete(UUID id)",
		),
	}
	manifest := []model.Contract{
		manifestContract("UserRepository", model.ContractOutputPort,
			"User save(User user)",             // identical → not in either delta
			"Optional<User> findById(UUID id)", // only in manifest → removed
			"void archive(UUID id)",            // only in manifest → removed
		),
	}

	actions := d.Diff(spec, manifest)

	require.Len(t, actions, 1)
	a := actions[0]
	require.Equal(t, diffvo.DiffActionModify, a.Type)
	// In spec but not in manifest: findByEmail, delete
	require.Equal(t, []string{"delete", "findByEmail"}, a.AddedMethods)
	// In manifest but not in spec: findById, archive
	require.Equal(t, []string{"archive", "findById"}, a.RemovedMethods)
}

// ─── MIXED ACTIONS ────────────────────────────────────────────────────────────

func TestContractDiffer_Diff_MixedSpecAndManifestProducesAllActionTypes(t *testing.T) {
	t.Parallel()

	d := newDiffer()

	spec := []model.SpecContract{
		specContract("NewContract", model.ContractInputPort, "void execute()"),
		specContract("SharedContract", model.ContractService, "void run()"),
	}
	manifest := []model.Contract{
		manifestContract("SharedContract", model.ContractService, "void run()"),
		manifestContract("ObsoleteContract", model.ContractService, "void old()"),
	}

	actions := d.Diff(spec, manifest)

	// NewContract → CREATE; SharedContract → no-op; ObsoleteContract → EXTRA.
	require.Len(t, actions, 2)

	createAction, ok := findAction(actions, "NewContract")
	require.True(t, ok, "expected CREATE action for NewContract")
	require.Equal(t, diffvo.DiffActionCreate, createAction.Type)

	extraAction, ok := findAction(actions, "ObsoleteContract")
	require.True(t, ok, "expected EXTRA action for ObsoleteContract")
	require.Equal(t, diffvo.DiffActionExtra, extraAction.Type)
}

// ─── LAYER DEFAULTS ───────────────────────────────────────────────────────────

func TestContractDiffer_Diff_CreateAndModifyHaveLayerZeroPlaceholder(t *testing.T) {
	t.Parallel()

	d := newDiffer()

	spec := []model.SpecContract{
		specContract("NewPort", model.ContractInputPort),
		specContract("ExistingService", model.ContractService, "void extra()"),
	}
	manifest := []model.Contract{
		manifestContract("ExistingService", model.ContractService /* no methods, so "extra" is added */),
	}

	actions := d.Diff(spec, manifest)

	require.Len(t, actions, 2)
	for _, a := range actions {
		if a.Type == diffvo.DiffActionCreate || a.Type == diffvo.DiffActionModify {
			require.Equal(t, 0, a.Layer, "CREATE/MODIFY placeholder Layer must be 0, got %d for %s", a.Layer, a.ContractName)
		}
	}
}

func TestContractDiffer_Diff_ExtraHasLayerNegativeOne(t *testing.T) {
	t.Parallel()

	d := newDiffer()

	spec := []model.SpecContract{}
	manifest := []model.Contract{
		manifestContract("GhostContract", model.ContractService),
	}

	actions := d.Diff(spec, manifest)

	require.Len(t, actions, 1)
	require.Equal(t, -1, actions[0].Layer)
}

// ─── DUPLICATE MANIFEST NAME (defensive) ──────────────────────────────────────

func TestContractDiffer_Diff_DuplicateManifestNameKeepsFirstOccurrence(t *testing.T) {
	t.Parallel()

	d := newDiffer()

	spec := []model.SpecContract{
		specContract("UserRepository", model.ContractOutputPort, "User save(User user)"),
	}
	// Two manifest entries with the same name — first should win.
	manifest := []model.Contract{
		manifestContract("UserRepository", model.ContractOutputPort, "User save(User user)"),
		manifestContract("UserRepository", model.ContractOutputPort, "void differentMethod()"),
	}

	// The first manifest entry matches the spec exactly → no-op.
	actions := d.Diff(spec, manifest)
	require.Empty(t, actions)
}
