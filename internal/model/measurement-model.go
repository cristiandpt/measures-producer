package actor

import (
	"time"
)

// MeasurementsModel represents a measurement entity in the system.
type MeasurementsModel struct {
	TimeStamp                               time.Time
	BloodPressureUnit                       string
	Systolic                                float64
	Diastolic                               float64
	MeanArterialPressure                    float64
	PulseRate                               float64
	BloodPressureMeasurementStatus          []string
	WeightUnit                              string
	HeightUnit                              string
	Weight                                  float64
	Height                                  float64
	BMI                                     float64
	BodyFatPercentage                       float64
	BasalMetabolism                         float64
	MusclePercentage                        float64
	MuscleMass                              float64
	FatFreeMass                             float64
	SoftLeanMass                            float64
	BodyWaterMass                           float64
	SkeletalMusclePercentage                float64
	VisceralFatLevel                        float64
	BodyAge                                 float64
	BodyFatPercentageStageEvaluation        float64
	SkeletalMusclePercentageStageEvaluation float64
	VisceralFatLevelStageEvaluation         float64
}
