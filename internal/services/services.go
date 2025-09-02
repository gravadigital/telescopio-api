package services

import (
	"errors"
	"time"

	"github.com/gravadigital/telescopio-api/internal/domain/event"
	"github.com/gravadigital/telescopio-api/internal/domain/participant"
	"github.com/gravadigital/telescopio-api/internal/storage/postgres"
	"github.com/gravadigital/telescopio-api/internal/validation"
)

// EventService maneja la lógica de negocio de eventos
type EventService struct {
	eventRepo postgres.EventRepository
	userRepo  postgres.UserRepository
	validator validation.EventValidation
}

// NewEventService crea una nueva instancia del servicio de eventos
func NewEventService(eventRepo postgres.EventRepository, userRepo postgres.UserRepository) *EventService {
	return &EventService{
		eventRepo: eventRepo,
		userRepo:  userRepo,
		validator: validation.EventValidation{},
	}
}

// CreateEventRequest representa una solicitud para crear un evento
type CreateEventRequest struct {
	Name        string    `json:"name" binding:"required"`
	Description string    `json:"description" binding:"required"`
	AuthorID    string    `json:"author_id" binding:"required"`
	StartDate   time.Time `json:"start_date" binding:"required"`
	EndDate     time.Time `json:"end_date" binding:"required"`
}

// CreateEvent crea un nuevo evento
func (s *EventService) CreateEvent(req CreateEventRequest) (*event.Event, error) {
	// Validaciones
	if err := s.validator.ValidateEventName(req.Name); err != nil {
		return nil, err
	}

	if err := s.validator.ValidateEventDescription(req.Description); err != nil {
		return nil, err
	}

	if err := validation.ValidateUUID(req.AuthorID, "author_id"); err != nil {
		return nil, err
	}

	if err := validation.ValidateDateRange(req.StartDate, req.EndDate); err != nil {
		return nil, err
	}

	// Verificar que el autor existe
	_, err := s.userRepo.GetByID(req.AuthorID)
	if err != nil {
		return nil, errors.New("author not found")
	}

	// Crear evento
	newEvent := event.NewEvent(
		req.Name,
		req.Description,
		req.AuthorID,
		req.StartDate,
		req.EndDate,
	)

	if err := s.eventRepo.Create(newEvent); err != nil {
		return nil, err
	}

	return newEvent, nil
}

// GetAllEvents obtiene todos los eventos
func (s *EventService) GetAllEvents() ([]*event.Event, error) {
	return s.eventRepo.GetAll()
}

// GetEventByID obtiene un evento por su ID
func (s *EventService) GetEventByID(id string) (*event.Event, error) {
	if err := validation.ValidateUUID(id, "event_id"); err != nil {
		return nil, err
	}

	return s.eventRepo.GetByID(id)
}

// RegisterParticipant registra un participante en un evento
func (s *EventService) RegisterParticipant(eventID, userID string) error {
	// Validaciones
	if err := validation.ValidateUUID(eventID, "event_id"); err != nil {
		return err
	}

	if err := validation.ValidateUUID(userID, "user_id"); err != nil {
		return err
	}

	// Verificar que el evento existe
	eventObj, err := s.eventRepo.GetByID(eventID)
	if err != nil {
		return errors.New("event not found")
	}

	// Verificar que el usuario exists
	_, err = s.userRepo.GetByID(userID)
	if err != nil {
		return errors.New("user not found")
	}

	// Verificar que el evento esté en la etapa correcta
	if eventObj.Stage != event.StageRegistration {
		return errors.New("event is not in registration stage")
	}

	// Registrar participante
	return s.eventRepo.AddParticipant(eventID, userID)
}

// UserService maneja la lógica de negocio de usuarios
type UserService struct {
	userRepo  postgres.UserRepository
	validator validation.UserValidation
}

// NewUserService crea una nueva instancia del servicio de usuarios
func NewUserService(userRepo postgres.UserRepository) *UserService {
	return &UserService{
		userRepo:  userRepo,
		validator: validation.UserValidation{},
	}
}

// CreateUserRequest representa una solicitud para crear un usuario
type CreateUserRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required"`
}

// CreateUser crea un nuevo usuario
func (s *UserService) CreateUser(req CreateUserRequest) (*participant.User, error) {
	// Validaciones
	if err := s.validator.ValidateUserName(req.Name); err != nil {
		return nil, err
	}

	if err := s.validator.ValidateUserEmail(req.Email); err != nil {
		return nil, err
	}

	// Verificar que el email no existe
	if _, err := s.userRepo.GetByEmail(req.Email); err == nil {
		return nil, errors.New("email already exists")
	}

	// Crear usuario
	newUser := participant.NewUser(req.Name, req.Email)

	if err := s.userRepo.Create(newUser); err != nil {
		return nil, err
	}

	return newUser, nil
}

// GetUserByEmail obtiene un usuario por su email
func (s *UserService) GetUserByEmail(email string) (*participant.User, error) {
	if err := s.validator.ValidateUserEmail(email); err != nil {
		return nil, err
	}

	return s.userRepo.GetByEmail(email)
}
