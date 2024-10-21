package control

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func (cs *ControlSystem) RouteGETWarnings(c *gin.Context) {
	// Get the warnings
	warnings := cs.generateTemperatureSensorWarnings()
	warnings = append(warnings, cs.generateHumiditySensorWarnings()...)
	warnings = append(warnings, cs.generateWeatherWarnings()...)
	c.JSON(http.StatusOK, warnings)
}

func (cs *ControlSystem) RoutePOSTDelayWatering(c *gin.Context) {
	// Get the delay, will look like /api/delay?unit=0
	unitStr := c.Query("unit")
	if unitStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"msg": "no unit to delay given",
		})
	}

	unit, err := strconv.Atoi(unitStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"msg": "invalid unit to delay, parse error",
		})
	}

	// Check if the unit is to be watered
	if _, ok := cs.systemTiming.NextWateringTime[uint(unit)]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"msg": "unit is not to be watered",
		})
	}

	// Now delay the watering
	cs.systemTiming.NextWateringTime[uint(unit)] = cs.systemTiming.NextWateringTime[uint(unit)].Add(60 * time.Minute)
	c.JSON(http.StatusOK, gin.H{
		"msg": fmt.Sprintf("unit %d has been delayed by 60 minutes", unit),
	})
}

func (cs *ControlSystem) RoutePOSTCancelWatering(c *gin.Context) {
	unitStr := c.Query("unit")
	if unitStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"msg": "no unit to stop watering given",
		})
	}

	unit, err := strconv.Atoi(unitStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"msg": "invalid unit to stop, parse error",
		})
	}

	if _, ok := cs.systemTiming.WateringUntilTime[uint(unit)]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"msg": "unit is not watering",
		})
	}

	// Otherwise cancel the thing
	if err = cs.HandleWateringOffEvent(uint(unit)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"msg": fmt.Sprintf("could not switch off watering for unit %d", unit),
		})
	}

}

// Schedule watering for now, route is /api/water-now?query=0
func (cs *ControlSystem) RoutePOSTWaterNow(c *gin.Context) {
	// In this one, set the watering time for 0 minute from now
	unitStr := c.Query("unit")
	if unitStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"msg": "no unit to start watering given",
		})
		return
	}
	unit, err := strconv.Atoi(unitStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"msg": "non-numeric unit given",
		})
		return
	}
	ok := false
	for _, unitConfig := range cs.systemConfig.RemoteUnitConfigs {
		if unitConfig.UnitNumber == uint(unit) {
			ok = true
			break
		}
	}
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"msg": "unit number does not exist",
		})
		return
	}
	cs.systemTiming.NextWateringTime[uint(unit)] = time.Now()
	c.JSON(http.StatusOK, gin.H{
		"msg": fmt.Sprintf("scheduled watering for unit %d for now", unit),
	})
}
