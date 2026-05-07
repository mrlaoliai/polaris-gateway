package webapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"polaris-gateway/internal/config"
	"polaris-gateway/internal/db"
	"polaris-gateway/internal/logger"
)

// AdminSettingsHandler handles GET and POST for /api/settings
func AdminSettingsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(config.AppConfig)
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			ListenAddr             string `json:"listen_addr"`
			InitialCooldownSeconds int    `json:"initial_cooldown_seconds"`
			MaxCooldownSeconds     int    `json:"max_cooldown_seconds"`
			FailureThreshold       int    `json:"failure_threshold"`
			FailureWindowSeconds   int    `json:"failure_window_seconds"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		_, err := db.DB().Exec("UPDATE sys_settings SET listen_addr=?, breaker_initial_cooldown_seconds=?, breaker_max_cooldown_seconds=?, breaker_failure_threshold=?, breaker_failure_window_seconds=? WHERE id=1",
			req.ListenAddr, req.InitialCooldownSeconds, req.MaxCooldownSeconds, req.FailureThreshold, req.FailureWindowSeconds)
		
		if err != nil {
			slog.Error("Failed to update settings", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		config.ReloadFromDB()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success"}`))
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// AdminNodesHandler handles CRUD for /api/nodes
func AdminNodesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		rows, err := db.DB().Query("SELECT id, platform, name, key_value, project_id, location, base_url, priority, cutoff_percent, budget, billing_start_date, is_enabled FROM sys_nodes ORDER BY platform, priority")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var nodes []map[string]interface{}
		for rows.Next() {
			var id, priority, isEnabled int
			var platform, name, key, projectID, location, baseURL, billingStartDate string
			var cutoffPercent, budget float64
			
			if err := rows.Scan(&id, &platform, &name, &key, &projectID, &location, &baseURL, &priority, &cutoffPercent, &budget, &billingStartDate, &isEnabled); err != nil {
				continue
			}
			// Mask key for security
			maskedKey := key
			if len(key) > 8 {
				maskedKey = key[:4] + "..." + key[len(key)-4:]
			} else {
				maskedKey = "***"
			}

			nodes = append(nodes, map[string]interface{}{
				"id":                 id,
				"platform":           platform,
				"name":               name,
				"key":                maskedKey, // Don't expose real key to frontend
				"project_id":         projectID,
				"location":           location,
				"base_url":           baseURL,
				"priority":           priority,
				"cutoff_percent":      cutoffPercent,
				"budget":              budget,
				"billing_start_date":  billingStartDate,
				"is_enabled":         isEnabled == 1,
			})
		}
		json.NewEncoder(w).Encode(nodes)
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Platform         string  `json:"platform"`
			Name             string  `json:"name"`
			Key              string  `json:"key"`
			ProjectID        string  `json:"project_id"`
			Location         string  `json:"location"`
			BaseURL          string  `json:"base_url"`
			Priority         int     `json:"priority"`
			CutoffPercent    float64 `json:"cutoff_percent"`
			Budget           float64 `json:"budget"`
			BillingStartDate string  `json:"billing_start_date"`
			IsEnabled        bool    `json:"is_enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		enabledInt := 0
		if req.IsEnabled {
			enabledInt = 1
		}

		_, err := db.DB().Exec(`
			INSERT INTO sys_nodes (platform, name, key_value, project_id, location, base_url, priority, cutoff_percent, budget, billing_start_date, is_enabled)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			req.Platform, req.Name, req.Key, req.ProjectID, req.Location, req.BaseURL, req.Priority, req.CutoffPercent, req.Budget, req.BillingStartDate, enabledInt)
		
		if err != nil {
			slog.Error("Failed to insert node", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		config.ReloadFromDB()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success"}`))
		return
	}

	if r.Method == http.MethodPut {
		var req struct {
			ID               int     `json:"id"`
			Platform         string  `json:"platform"`
			Name             string  `json:"name"`
			Key              string  `json:"key"`
			ProjectID        string  `json:"project_id"`
			Location         string  `json:"location"`
			BaseURL          string  `json:"base_url"`
			Priority         int     `json:"priority"`
			CutoffPercent    float64 `json:"cutoff_percent"`
			Budget           float64 `json:"budget"`
			BillingStartDate string  `json:"billing_start_date"`
			IsEnabled        bool    `json:"is_enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		enabledInt := 0
		if req.IsEnabled {
			enabledInt = 1
		}

		// Only update key if it wasn't masked
		if !strings.Contains(req.Key, "...") && req.Key != "***" && req.Key != "" {
			_, err := db.DB().Exec(`
				UPDATE sys_nodes SET platform=?, name=?, key_value=?, project_id=?, location=?, base_url=?, priority=?, cutoff_percent=?, budget=?, billing_start_date=?, is_enabled=?
				WHERE id=?`,
				req.Platform, req.Name, req.Key, req.ProjectID, req.Location, req.BaseURL, req.Priority, req.CutoffPercent, req.Budget, req.BillingStartDate, enabledInt, req.ID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			_, err := db.DB().Exec(`
				UPDATE sys_nodes SET platform=?, name=?, project_id=?, location=?, base_url=?, priority=?, cutoff_percent=?, budget=?, billing_start_date=?, is_enabled=?
				WHERE id=?`,
				req.Platform, req.Name, req.ProjectID, req.Location, req.BaseURL, req.Priority, req.CutoffPercent, req.Budget, req.BillingStartDate, enabledInt, req.ID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		
		config.ReloadFromDB()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success"}`))
		return
	}

	if r.Method == http.MethodDelete {
		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			http.Error(w, "missing id parameter", http.StatusBadRequest)
			return
		}
		id, _ := strconv.Atoi(idStr)
		_, err := db.DB().Exec("DELETE FROM sys_nodes WHERE id=?", id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		config.ReloadFromDB()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success"}`))
		return
	}
	
	
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// AdminLogsHandler returns the tail of the polaris-gateway.log file
func AdminLogsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if logger.LogFile == nil {
		w.Write([]byte("No log file configured or polaris-gateway.log not found.\n"))
		return
	}

	info, err := logger.LogFile.Stat()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Read last 50KB maximum
	var size int64 = 50 * 1024
	if info.Size() < size {
		size = info.Size()
	}

	buf := make([]byte, size)
	_, err = logger.LogFile.ReadAt(buf, info.Size()-size)
	if err != nil && err != io.EOF {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(buf)
}
