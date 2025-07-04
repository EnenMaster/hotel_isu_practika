package auth

import "net/http"

// Is возвращает true, если в сессии роль совпадает с указанной.
func Is(r *http.Request, role string) bool {
    if s, ok := Get(r); ok {
        return s.Role == role
    }
    return false
}
