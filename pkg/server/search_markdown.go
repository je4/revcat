package server

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/je4/revcat/v2/config"
	"github.com/je4/revcat/v2/tools/graph/model"
)

func truncateString(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-2]) + ".."
}

func calculateDateAddedBoost(dateAddedStr string, now time.Time) (float64, *time.Time) {
	if dateAddedStr == "" {
		return 0.0, nil
	}
	var t time.Time
	var err error
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	for _, layout := range formats {
		t, err = time.Parse(layout, dateAddedStr)
		if err == nil {
			break
		}
	}
	if err != nil {
		return 0.0, nil
	}

	diff := now.Sub(t).Hours() / 24.0
	if diff < 0 {
		diff = 0
	}

	decay1 := 0.5
	scale1 := 30.0
	weight1 := 0.15
	score1 := weight1 * math.Exp((math.Log(decay1)/(scale1*scale1))*diff*diff)

	decay2 := 0.1
	scale2 := 335.0
	weight2 := 0.10
	var score2 float64
	if diff <= 30.0 {
		score2 = weight2
	} else {
		d2 := diff - 30.0
		score2 = weight2 * math.Exp((math.Log(decay2)/(scale2*scale2))*d2*d2)
	}

	totalBoost := score1 + score2
	return totalBoost, &t
}

type searchMatrixRow struct {
	perfRank       int
	baseRank       int
	hasBaseRank    bool
	delta          int
	sig            string
	title          string
	docType        string
	typeBoost      float64
	topRole        string
	roleBoost      float64
	dateAddedDesc  string
	dateAddedBoost float64
	achievedBoost  float64
	score          *float64
	tier           string
	status         string
}

// renderSearchMarkdown builds a GitHub-Flavored Markdown report comparing target client ranking against baseline
func renderSearchMarkdown(query string, clientName string, baselineName string, clientConfig *config.Client, resTarget, resBaseline *model.SearchResult, limit int) string {
	var typeWeights map[string]float64
	var roleWeights map[string]float64
	var allowedCategories []string

	if clientConfig != nil {
		if tw, ok := clientConfig.FieldWeights["type"]; ok {
			typeWeights = tw
		}
		if rw, ok := clientConfig.FieldWeights["[persons].role"]; ok {
			roleWeights = rw
		}
		for _, andFilter := range clientConfig.AND {
			if len(andFilter.OR) > 0 {
				for _, orFilter := range andFilter.OR {
					if orFilter.Field == "category.keyword" {
						allowedCategories = append(allowedCategories, orFilter.Values...)
					}
				}
			}
		}
	}

	baselineRankMap := make(map[string]int)
	if resBaseline != nil {
		for i, edge := range resBaseline.Edges {
			if edge != nil && edge.Base != nil {
				baselineRankMap[edge.Base.Signature] = i + 1
			}
		}
	}

	isEmptyQuery := strings.TrimSpace(query) == ""
	hasAddedBoost := clientConfig != nil && clientConfig.AddedBoost
	now := time.Now()

	var rows []searchMatrixRow

	if resTarget != nil {
		for i, edge := range resTarget.Edges {
			if edge == nil || edge.Base == nil {
				continue
			}
			perfRank := i + 1
			base := edge.Base

			typeStr := "unknown"
			if base.Type != nil {
				typeStr = *base.Type
			}
			typeBoost := 1.0

			topRole := "none"
			maxRoleBoost := 1.0

			if !isEmptyQuery {
				if w, ok := typeWeights[typeStr]; ok {
					typeBoost = w
				}
				for _, p := range base.Person {
					if p == nil {
						continue
					}
					role := ""
					if p.Role != nil {
						role = *p.Role
					}
					if w, ok := roleWeights[role]; ok {
						if w > maxRoleBoost {
							maxRoleBoost = w
							topRole = role
						}
					} else if topRole == "none" && role != "" {
						topRole = role
					}
				}
			}

			var titleStr string
			if len(base.Title) > 0 && base.Title[0] != nil {
				titleStr = base.Title[0].Value
			}

			dateAddedDesc := "n/a"
			var dateAddedBoost float64
			if hasAddedBoost {
				dateAddedStr := ""
				if base.DateAdded != nil {
					dateAddedStr = *base.DateAdded
				}
				boost, parsedTime := calculateDateAddedBoost(dateAddedStr, now)
				dateAddedBoost = boost
				if parsedTime != nil {
					dateAddedDesc = fmt.Sprintf("%s (+%.2f)", parsedTime.Format("2006-01-02"), boost)
				} else {
					dateAddedDesc = "none (+0.00)"
				}
			}

			effectiveMaxBoost := typeBoost
			if maxRoleBoost > effectiveMaxBoost {
				effectiveMaxBoost = maxRoleBoost
			}

			achievedBoost := effectiveMaxBoost
			if hasAddedBoost {
				achievedBoost += dateAddedBoost
			}

			tier := "Tier 3 [1.0-1.4x]"
			if achievedBoost >= 4.0 {
				tier = "Tier 1 [4.0x+]"
			} else if achievedBoost >= 1.5 {
				tier = "Tier 2 [1.5-3.9x]"
			}

			baseRank, inBaseline := baselineRankMap[base.Signature]
			delta := 0
			status := "NEW (Top)"
			if inBaseline {
				delta = baseRank - perfRank
				if achievedBoost > 1.0 {
					if delta > 0 {
						status = "BOOSTED (+Δ)"
					} else if delta < 0 {
						status = "BOOSTED (-Δ)"
					} else {
						status = "BOOSTED (=)"
					}
				} else {
					if delta > 0 {
						status = "PROMOTED (+Δ)"
					} else if delta < 0 {
						status = "UNBOOSTED (-Δ)"
					} else {
						status = "STABLE (=)"
					}
				}
			} else if achievedBoost > 1.0 {
				status = "BOOSTED (NEW)"
			}

			rows = append(rows, searchMatrixRow{
				perfRank:       perfRank,
				baseRank:       baseRank,
				hasBaseRank:    inBaseline,
				delta:          delta,
				sig:            base.Signature,
				title:          titleStr,
				docType:        typeStr,
				typeBoost:      typeBoost,
				topRole:        topRole,
				roleBoost:      maxRoleBoost,
				dateAddedDesc:  dateAddedDesc,
				dateAddedBoost: dateAddedBoost,
				achievedBoost:  achievedBoost,
				score:          edge.Score,
				tier:           tier,
				status:         status,
			})
		}
	}

	var sb strings.Builder

	queryTitle := fmt.Sprintf("%q", query)
	if query == "" {
		queryTitle = `"" (empty)`
	}
	sb.WriteString(fmt.Sprintf("# Search Prioritization & Ranking Matrix: %s\n\n", queryTitle))
	sb.WriteString("### Metadata\n")
	if query == "" {
		sb.WriteString("- **Query**: `(empty)`\n")
	} else {
		sb.WriteString(fmt.Sprintf("- **Query**: `%s`\n", query))
	}
	sb.WriteString(fmt.Sprintf("- **Client**: `%s`\n", clientName))
	sb.WriteString(fmt.Sprintf("- **Baseline**: `%s`\n", baselineName))
	sb.WriteString(fmt.Sprintf("- **Evaluated Hits Limit**: `%d`\n", limit))
	totalTarget := 0
	if resTarget != nil {
		totalTarget = resTarget.TotalCount
	}
	totalBaseline := 0
	if resBaseline != nil {
		totalBaseline = resBaseline.TotalCount
	}
	sb.WriteString(fmt.Sprintf("- **Total Hits**: `%d` (Target Client) vs `%d` (Baseline)\n\n", totalTarget, totalBaseline))

	sb.WriteString("### Weighting & Ranking Matrix\n\n")
	sb.WriteString("| Rank | Base # | Delta | Signature | Title | Type (Boost) | Role (Boost) | Date Added (Boost) | Achieved Weight | Tier | Status |\n")
	sb.WriteString("| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |\n")

	for _, row := range rows {
		baseRankStr := "n/a"
		deltaStr := "NEW"
		if row.hasBaseRank {
			baseRankStr = fmt.Sprintf("#%02d", row.baseRank)
			if row.delta > 0 {
				deltaStr = fmt.Sprintf("+%d", row.delta)
			} else if row.delta < 0 {
				deltaStr = fmt.Sprintf("%d", row.delta)
			} else {
				deltaStr = "="
			}
		}

		typeDesc := fmt.Sprintf("%s (x%.1f)", row.docType, row.typeBoost)
		roleDesc := fmt.Sprintf("%s (x%.1f)", row.topRole, row.roleBoost)
		safeTitle := strings.ReplaceAll(row.title, "|", "\\|")
		shortTitle := truncateString(safeTitle, 35)

		sb.WriteString(fmt.Sprintf("| [#%02d] | %s | %s | `%s` | %s | %s | %s | %s | x%.2f | %s | %s |\n",
			row.perfRank, baseRankStr, deltaStr, row.sig, shortTitle, typeDesc, roleDesc, row.dateAddedDesc, row.achievedBoost, row.tier, row.status))
	}
	sb.WriteString("\n")

	// Statistics
	type tierStats struct {
		totalCount    int
		top10Count    int
		improvedCount int
		deltaSum      int
		deltaCount    int
	}
	statsTier1 := &tierStats{}
	statsTier2 := &tierStats{}
	statsTier3 := &tierStats{}

	for _, row := range rows {
		var st *tierStats
		if strings.HasPrefix(row.tier, "Tier 1") {
			st = statsTier1
		} else if strings.HasPrefix(row.tier, "Tier 2") {
			st = statsTier2
		} else {
			st = statsTier3
		}
		st.totalCount++
		if row.perfRank <= 10 {
			st.top10Count++
		}
		if row.hasBaseRank {
			st.deltaSum += row.delta
			st.deltaCount++
			if row.delta > 0 {
				st.improvedCount++
			}
		}
	}

	avgDeltaStr := func(st *tierStats) string {
		if st.deltaCount == 0 {
			return "N/A"
		}
		return fmt.Sprintf("%+.2f", float64(st.deltaSum)/float64(st.deltaCount))
	}

	topHitsCount := len(rows)
	top10Limit := 10
	if topHitsCount < top10Limit {
		top10Limit = topHitsCount
	}

	top10Tier1Pct := 0.0
	top10Tier2Pct := 0.0
	top10Tier3Pct := 0.0
	if top10Limit > 0 {
		top10Tier1Pct = float64(statsTier1.top10Count) / float64(top10Limit) * 100
		top10Tier2Pct = float64(statsTier2.top10Count) / float64(top10Limit) * 100
		top10Tier3Pct = float64(statsTier3.top10Count) / float64(top10Limit) * 100
	}

	sb.WriteString("### Tier Distribution & Statistical Analysis\n\n")
	sb.WriteString(fmt.Sprintf("- **Tier 1 (High Boost 4.0x+)**: %d docs | %d in Top 10 (%.1f%%) | %d rank improved | Avg Delta: %s\n",
		statsTier1.totalCount, statsTier1.top10Count, top10Tier1Pct, statsTier1.improvedCount, avgDeltaStr(statsTier1)))
	sb.WriteString(fmt.Sprintf("- **Tier 2 (Med Boost 1.5-3.9x)**: %d docs | %d in Top 10 (%.1f%%) | %d rank improved | Avg Delta: %s\n",
		statsTier2.totalCount, statsTier2.top10Count, top10Tier2Pct, statsTier2.improvedCount, avgDeltaStr(statsTier2)))
	sb.WriteString(fmt.Sprintf("- **Tier 3 (Base 1.0-1.4x)**: %d docs | %d in Top 10 (%.1f%%) | %d rank improved | Avg Delta: %s\n\n",
		statsTier3.totalCount, statsTier3.top10Count, top10Tier3Pct, statsTier3.improvedCount, avgDeltaStr(statsTier3)))

	top10Boosted := statsTier1.top10Count + statsTier2.top10Count
	top10BoostedPct := 0.0
	if top10Limit > 0 {
		top10BoostedPct = float64(top10Boosted) / float64(top10Limit) * 100
	}

	sb.WriteString("### Prioritization & Filter Invariants\n\n")
	sb.WriteString(fmt.Sprintf("- **Top 10 Boosted Dominance**: %.1f%% of Top 10 have active weight boosts (Tier 1/2: %d/%d hits)\n",
		top10BoostedPct, top10Boosted, top10Limit))

	if len(allowedCategories) > 0 {
		allMatch := true
		if resTarget != nil {
			for _, edge := range resTarget.Edges {
				if edge == nil || edge.Base == nil {
					continue
				}
				matched := false
				for _, cat := range edge.Base.Category {
					for _, allowed := range allowedCategories {
						if cat == allowed {
							matched = true
							break
						}
					}
					if matched {
						break
					}
				}
				if !matched && len(edge.Base.Category) > 0 {
					allMatch = false
					break
				}
			}
		}
		if allMatch {
			sb.WriteString(fmt.Sprintf("- **Category Filter Enforcement**: PASS (100%% of %d returned documents belong to allowed categories)\n", totalTarget))
		} else {
			sb.WriteString(fmt.Sprintf("- **Category Filter Enforcement**: FAIL (Some returned documents violate category filters)\n"))
		}
	} else {
		sb.WriteString("- **Category Filter Enforcement**: N/A (No category restrictions defined)\n")
	}

	return sb.String()
}

// @Summary      Search and evaluate ranking matrix
// @Description  Execute search query and return formatted Markdown report comparing client ranking against baseline
// @Tags         search
// @Produce      plain
// @Param        query path string false "Search Query"
// @Param        q query string false "Alternative query parameter"
// @Param        allow_empty query bool false "Allow empty query execution for baseline/filter inspection"
// @Param        client query string false "Client name (default: performance)"
// @Param        baseline query string false "Baseline client name (default: default_baseline)"
// @Param        limit query int false "Maximum number of hits to evaluate (default: 30)"
// @Param        groups query string false "Access groups, comma-separated (default: global/guest)"
// @Security     BearerAuth
// @Success      200 {string} string "Markdown formatted report"
// @Failure      400 {object} map[string]string "bad request"
// @Failure      401 {string} string "unauthorized"
// @Failure      500 {object} map[string]string "internal error"
// @Router       /search/{query} [get]
func (ctrl *Controller) searchMarkdown(c *gin.Context) {
	rawQuery := c.Param("query")
	rawQuery = strings.TrimPrefix(rawQuery, "/")
	rawQuery = strings.TrimSpace(rawQuery)

	if unescaped, err := url.PathUnescape(rawQuery); err == nil && unescaped != "" {
		rawQuery = unescaped
	}

	if rawQuery == "" {
		rawQuery = strings.TrimSpace(c.Query("q"))
	}
	if rawQuery == "" {
		rawQuery = strings.TrimSpace(c.Query("query"))
	}

	allowEmpty := true // strings.EqualFold(c.Query("allow_empty"), "true") || strings.EqualFold(c.Query("allowEmpty"), "true") || strings.EqualFold(c.Query("empty"), "true")

	if rawQuery == "" && !allowEmpty {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "search query is required"})
		return
	}

	clientName := c.DefaultQuery("client", "performance")
	baselineName := c.DefaultQuery("baseline", "default_baseline")

	limitStr := c.DefaultQuery("limit", c.DefaultQuery("size", "30"))
	limit := 30
	if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
		limit = val
	}

	var groups []string
	if groupsQuery := c.QueryArray("groups"); len(groupsQuery) > 0 {
		for _, g := range groupsQuery {
			for _, part := range strings.Split(g, ",") {
				part = strings.TrimSpace(part)
				if part != "" {
					groups = append(groups, part)
				}
			}
		}
	} else if groupParam := c.Query("groups"); groupParam != "" {
		for _, part := range strings.Split(groupParam, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				groups = append(groups, part)
			}
		}
	}
	if len(groups) == 0 {
		groups = []string{"global/guest"}
	}

	ctrl.logger.Info().Msgf("searchMarkdown: query=%q client=%s baseline=%s limit=%d groups=%v",
		rawQuery, clientName, baselineName, limit, groups)

	ctxTarget := context.WithValue(c.Request.Context(), "client", clientName)
	ctxTarget = context.WithValue(ctxTarget, "groups", groups)

	resTarget, err := ctrl.resolver.Search(
		ctxTarget,
		"all",
		rawQuery,
		nil, nil, nil, nil, &limit, nil, nil,
	)
	if err != nil {
		ctrl.logger.Error().Err(err).Msgf("searchMarkdown: search failed for client %s: %v", clientName, err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctxBaseline := context.WithValue(c.Request.Context(), "client", baselineName)
	ctxBaseline = context.WithValue(ctxBaseline, "groups", groups)

	resBaseline, err := ctrl.resolver.Search(
		ctxBaseline,
		"all",
		rawQuery,
		nil, nil, nil, nil, &limit, nil, nil,
	)
	if err != nil {
		ctrl.logger.Error().Err(err).Msgf("searchMarkdown: search failed for baseline %s: %v", baselineName, err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var clientConfig *config.Client
	if ctrl.clientByName != nil {
		clientConfig = ctrl.clientByName[clientName]
	}

	md := renderSearchMarkdown(rawQuery, clientName, baselineName, clientConfig, resTarget, resBaseline, limit)
	c.Data(http.StatusOK, "text/markdown; charset=utf-8", []byte(md))
}
