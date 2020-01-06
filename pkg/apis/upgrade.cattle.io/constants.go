package upgrade

import "path"

const (
	// LabelController is the name of the upgrade controller.
	LabelController = GroupName + `/controller`

	// LabelNode is the node being upgraded.
	LabelNode = GroupName + `/node`

	// LabelPlan is the plan being applied.
	LabelPlan = GroupName + `/plan`

	// LabelPlanSuffix is used for composing labels specific to a plan.
	LabelPlanSuffix = `plan.` + GroupName
)

func LabelPlanName(name string) string {
	return path.Join(LabelPlanSuffix, name)
}
