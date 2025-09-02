package api

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/vultisig/pluginagent/types"
)

type WebSocketMessage struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

type SubscriptionRequest struct {
	Channel  string `json:"channel"`
	LastSeen *int64 `json:"last_seen,omitempty"`
}

type EventMessage struct {
	ID        int64                 `json:"id"`
	PublicKey *string               `json:"public_key"`
	PolicyID  *string               `json:"policy_id,omitempty"`
	EventType types.SystemEventType `json:"event_type"`
	EventData json.RawMessage       `json:"event_data"`
	CreatedAt time.Time             `json:"created_at"`
}

type ClientConnection struct {
	ws            *websocket.Conn
	subscriptions map[string]bool
	mutex         sync.RWMutex
	replayMutex   sync.Mutex
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (s *Server) GetEvents(c echo.Context) error {
	s.logger.Info("GetEvents WebSocket upgrade")
	
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		s.logger.WithError(err).Error("WebSocket upgrade failed")
		return err
	}
	
	s.logger.Info("WebSocket connected")
	defer func() {
		ws.Close()
		s.logger.Info("WebSocket disconnected")
	}()

	client := &ClientConnection{
		ws:            ws,
		subscriptions: make(map[string]bool),
	}

	for {
		var msg WebSocketMessage
		err := ws.ReadJSON(&msg)
		if err != nil {
			s.logger.WithError(err).Debug("WebSocket receive error")
			return nil
		}

		switch msg.Type {
		case "subscribe":
			var subReq SubscriptionRequest
			data, _ := json.Marshal(msg.Data)
			if err := json.Unmarshal(data, &subReq); err != nil {
				s.sendError(ws, "invalid subscription request")
				continue
			}

			if subReq.Channel == "system_events" {
				s.handleSystemEventsSubscription(client, subReq, c)
			} else {
				s.sendError(ws, "unknown channel")
			}

		default:
			s.sendError(ws, "unknown message type")
		}
	}
}

func (s *Server) handleSystemEventsSubscription(client *ClientConnection, req SubscriptionRequest, ctx echo.Context) {
	client.mutex.Lock()
	client.subscriptions["system_events"] = true
	client.mutex.Unlock()

	go func() {
		client.replayMutex.Lock()
		defer client.replayMutex.Unlock()

		if req.LastSeen != nil {
			lastSeenTime := time.Unix(*req.LastSeen, 0)
			events, err := s.db.GetEventsAfterTimestamp(ctx.Request().Context(), lastSeenTime)
			if err != nil {
				s.logger.WithError(err).Error("Failed to get historical events")
				s.sendError(client.ws, "failed to get historical events")
				return
			}

			for _, event := range events {
				eventMsg := s.convertToEventMessage(event)
				if err := s.sendEvent(client.ws, eventMsg); err != nil {
					s.logger.WithError(err).Debug("Failed to send historical event")
					return
				}
			}
		}

		s.sendMessage(client.ws, WebSocketMessage{
			Type: "subscription_confirmed",
			Data: map[string]string{"channel": "system_events"},
		})
	}()
}

func (s *Server) convertToEventMessage(event types.SystemEvent) EventMessage {
	var policyIDStr *string
	if event.PolicyID != nil {
		policyStr := event.PolicyID.String()
		policyIDStr = &policyStr
	}

	return EventMessage{
		ID:        event.ID,
		PublicKey: event.PublicKey,
		PolicyID:  policyIDStr,
		EventType: event.EventType,
		EventData: json.RawMessage(event.EventData),
		CreatedAt: event.CreatedAt,
	}
}

func (s *Server) sendEvent(ws *websocket.Conn, event EventMessage) error {
	return s.sendMessage(ws, WebSocketMessage{
		Type: "event",
		Data: event,
	})
}

func (s *Server) sendMessage(ws *websocket.Conn, msg WebSocketMessage) error {
	return ws.WriteJSON(msg)
}

func (s *Server) sendError(ws *websocket.Conn, errorMsg string) {
	s.sendMessage(ws, WebSocketMessage{
		Type: "error",
		Data: map[string]string{"message": errorMsg},
	})
}
