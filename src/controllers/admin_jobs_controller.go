package controllers

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/jobs"
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TriggerCompleteProgram - enqueue a complete-program task to run after delaySec seconds (default 5s)
// TriggerCompleteProgram godoc
// @Summary      Enqueue program completion job (test)
// @Description  Enqueue a complete-program task to run after delaySec seconds. Requires Asynq (Redis) configured. Use for testing scheduling behavior.
// @Tags         programs
// @Accept       json
// @Produce      json
// @Param        id      path      string  true  "Program ID"
// @Param        delaySec  query   int     false "Delay in seconds"  default(5)
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  models.ErrorResponse
// @Failure      503  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /programs/{id}/trigger-complete [post]
func TriggerCompleteProgram(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "program id required"})
	}

	// find program to get name
	pid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid program id"})
	}
	var prg struct {
		Name *string `bson:"name"`
	}
	if err := DB.ProgramCollection.FindOne(context.TODO(), bson.M{"_id": pid}).Decode(&prg); err != nil {
		// if not found, continue but programName will be empty
		// return error? choose to return 404
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "program not found"})
	}
	programName := ""
	if prg.Name != nil {
		programName = *prg.Name
	}

	delaySec := 5
	if q := c.Query("delaySec"); q != "" {
		if v, err := strconv.Atoi(q); err == nil && v >= 0 {
			delaySec = v
		}
	}

	// build task with program name
	task, err := jobs.NewCompleteProgramTaskWithName(id, programName)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	if DB.AsynqClient == nil {
		return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{"error": "asynq client not initialized"})
	}

	// Enqueue to run after delaySec seconds
	_, err = DB.AsynqClient.Enqueue(task, asynq.ProcessIn(time.Duration(delaySec)*time.Second), asynq.TaskID("trigger-complete-"+id+"-"+time.Now().Format("20060102150405")))
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "enqueued", "delaySec": delaySec, "programName": programName})
}

// RunCompleteProgramNow - directly runs the complete handler in-process for quick testing (no Redis required)
// RunCompleteProgramNow godoc
// @Summary      Execute program completion now (in-process)
// @Description  Run the complete-program handler synchronously in-process for quick testing. This does not require Redis/Asynq and will execute the same logic as the background worker.
// @Tags         programs
// @Accept       json
// @Produce      json
// @Param        id      path      string  true  "Program ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /programs/{id}/run-complete-now [post]
func RunCompleteProgramNow(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "program id required"})
	}
	// find program name to include in payload/response
	pid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid program id"})
	}
	var prg struct {
		Name *string `bson:"name"`
	}
	if err := DB.ProgramCollection.FindOne(context.TODO(), bson.M{"_id": pid}).Decode(&prg); err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "program not found"})
	}
	programName := ""
	if prg.Name != nil {
		programName = *prg.Name
	}

	payload := jobs.ProgramPayload{ProgramID: id, ProgramName: programName}
	b, _ := json.Marshal(payload)
	// create a fake asynq.Task
	t := asynq.NewTask(jobs.TypeCompleteProgram, b)

	// Call handler directly
	if err := jobs.HandleCompleteProgramTask(context.TODO(), t); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "executed", "programName": programName})
}
