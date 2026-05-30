// This file has been refactored and modularized
// The implementation has been moved to the user package for better organization
//
// All UserService functionality is now available through:
//   - internal/app/services/user/service.go         - Main service and constructor
//   - internal/app/services/user/crud.go           - CRUD operations
//   - internal/app/services/user/auth.go           - Authentication methods
//   - internal/app/services/user/password.go       - Password management
//   - internal/app/services/user/activities.go     - User activity analytics
//   - internal/app/services/user/metrics_calculator.go - Business metrics calculation
//   - internal/app/services/user/business_context.go   - Efficient context mapping
//
// Import and usage remains the same, but with improved:
//   ✅ Proper observability logging
//   ✅ O(1) context lookups instead of switch statements
//   ✅ Better separation of concerns
//   ✅ Enhanced maintainability

package user

// Service type alias for backward compatibility
type Service = UserService

// NewService is an alias for backward compatibility
var NewService = NewUserService
