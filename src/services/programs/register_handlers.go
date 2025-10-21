package programs

import (
    "os"
    "strings"

    emailpkg "Backend-Bluelock-007/src/services/programs/email"
    "github.com/hibiken/asynq"
)

func RegisterProgramHandlers(mux *asynq.ServeMux) error {
    sender, err := emailpkg.NewSMTPSenderFromEnv()
    if err != nil { return err }

    base := strings.TrimRight(os.Getenv("APP_BASE_URL"), "/")
    registerURL := func(programID string) string {
        return base + "/Student/Programs/" + programID
    }

    // ส่ง resolver + prefix generator จากแพ็กเกจ programs เข้าสู่ email handler
    mux.HandleFunc(
        emailpkg.TypeNotifyOpenProgram,
        emailpkg.HandleNotifyOpenProgram(sender, registerURL, GetProgramByID, GenerateStudentCodeFilter),
    )
    return nil
}
