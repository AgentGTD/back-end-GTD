package encoreapp

import (
    "context"
    "errors"
    "log"
    "sync"

    firebase "firebase.google.com/go/v4"
    "firebase.google.com/go/v4/auth"
    "google.golang.org/api/option"
)

var (
    firebaseApp  *firebase.App
    firebaseAuth *auth.Client
    initOnce     sync.Once
)

// Call this once at startup (e.g. in main or an init function)
func InitFirebase() {
    initOnce.Do(func() {
        opt := option.WithCredentialsFile("back-end-GTD/dev/flowdo-aa2dc-firebase-adminsdk-fbsvc-81408554d0.json")
        app, err := firebase.NewApp(context.Background(), nil, opt)
        if err != nil {
            log.Fatalf("error initializing firebase app: %v", err)
        }
        firebaseApp = app
        firebaseAuth, err = app.Auth(context.Background())
        if err != nil {
            log.Fatalf("error initializing firebase auth: %v", err)
        }
    })
}

// Returns the Firebase user info if the token is valid, else error.
func getFirebaseUser(ctx context.Context, idToken string) (*auth.Token, error) {
    InitFirebase() // Ensure Firebase is initialized
    token, err := firebaseAuth.VerifyIDToken(ctx, idToken)
    if err != nil {
        return nil, errors.New("invalid or expired Firebase ID token")
    }
    return token, nil
}