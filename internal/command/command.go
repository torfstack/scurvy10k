package command

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"io"
	"net/http"
	"os"
	"scurvy10k/internal/models"
	"scurvy10k/internal/utils"
	sqlc "scurvy10k/sql/gen"
	"slices"
	"strings"

	"github.com/rs/zerolog/log"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/cmdroute"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
)

const (
	BaseUrl  = "https://true.torfstack.com/"
	DebtsUrl = BaseUrl + "api/debt"
)

var (
	torfstackId discord.UserID
	channelId   discord.ChannelID
	messageId   discord.MessageID
)

func Setup() {
	userId, err := discord.ParseSnowflake("263352209654153236")
	if err != nil {
		log.Error().Msgf("cannot parse torfstack id: %s", err)
		return
	}
	torfstackId = discord.UserID(userId)

	conn, err := utils.GetConnection(context.Background(), utils.DefaultConfig())
	if err != nil {
		log.Error().Msgf("cannot get db connection: %s", err)
		return
	}
	defer func(conn *pgx.Conn, ctx context.Context) {
		_ = conn.Close(ctx)
	}(conn, context.Background())

	q := sqlc.New(conn)
	setup, err := q.GetBotSetup(context.Background())
	if err != nil {
		log.Error().Msgf("cannot get bot setup: %s", err)
		return
	}
	messageIdSnowflake, err := discord.ParseSnowflake(setup.MessageID)
	if err != nil {
		log.Error().Msgf("cannot parse message id: %s", err)
		return
	}
	messageId = discord.MessageID(messageIdSnowflake)
	channelIdSnowflake, err := discord.ParseSnowflake(setup.ChannelID)
	if err != nil {
		log.Error().Msgf("cannot parse channel id: %s", err)
		return
	}
	channelId = discord.ChannelID(channelIdSnowflake)
}

func AddDebt(s *state.State) func(ctx context.Context, data cmdroute.CommandData) *api.InteractionResponseData {
	return func(ctx context.Context, data cmdroute.CommandData) *api.InteractionResponseData {
		options := data.Options
		name := options.Find("name").String()
		amount := options.Find("amount").String()

		res, err := http.Post(DebtsUrl+"/"+name+"/"+amount, "application/json", nil)
		if err != nil {
			log.Error().Msgf("could not send debt post request: %s", err)
			return ephemeralMessage("Could not update debt")
		}
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(res.Body)
		if res.StatusCode != http.StatusOK {
			log.Error().Msgf("debt post request was not successful: %s", res.Status)
			return ephemeralMessage("Could not update debt")
		}
		if channelId != discord.NullChannelID && messageId != discord.NullMessageID {
			updateDebtsMessage(s)
		}
		return &api.InteractionResponseData{
			Content: option.NewNullableString(fmt.Sprintf("Added %v to %v", amount, name)),
			Flags:   discord.EphemeralMessage,
		}
	}
}

func SetChannel(s *state.State) func(ctx context.Context, data cmdroute.CommandData) *api.InteractionResponseData {
	return func(ctx context.Context, data cmdroute.CommandData) *api.InteractionResponseData {
		if data.Event.SenderID() != torfstackId {
			log.Error().Msgf("cannot set channel: not torfstack, got %v", data.Event.SenderID())
			return ephemeralMessage("You are not allowed to set the channel, ask Torfstack!")
		}
		options := data.Options
		var err error
		cId, err := options.Find("channel_id").SnowflakeValue()
		if err != nil {
			log.Error().Msgf("cannot get channel_id: %s", err)
			return ephemeralMessage("Could not set channel")
		}
		channelId = discord.ChannelID(cId)
		debts, err := getDebts()
		if err != nil {
			log.Error().Msgf("cannot get debts: %s", err)
			return ephemeralMessage("Could not get debts")
		}
		m, err := s.SendMessage(discord.ChannelID(channelId), "", *transformDebtsToEmbed(debts))
		if err != nil {
			log.Error().Msgf("cannot send message: %s", err)
			return ephemeralMessage("Could not send message")
		}
		messageId = m.ID
		conn, err := utils.GetConnection(context.Background(), utils.DefaultConfig())
		if err != nil {
			log.Error().Msgf("cannot get db connection: %s", err)
			return ephemeralMessage("Could not get db connection")
		}
		defer func(conn *pgx.Conn, ctx context.Context) {
			_ = conn.Close(ctx)
		}(conn, ctx)
		q := sqlc.New(conn)
		_, err = q.PutBotSetup(ctx, sqlc.PutBotSetupParams{
			ChannelID: channelId.String(),
			MessageID: messageId.String(),
		})
		if err != nil {
			log.Error().Msgf("cannot put bot setup: %s", err)
			return ephemeralMessage("Could not put bot setup")
		}
		return ephemeralMessage("Channel set successfully")
	}
}

func getDebts() (*models.AllDebtsResponse, error) {
	res, err := http.Get(DebtsUrl)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(res.Body)
	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var debts models.AllDebtsResponse
	if err = json.Unmarshal(bytes, &debts); err != nil {
		log.Error().Msgf("cannot unmarshal debts: %s", err)
		return nil, err
	}
	return &debts, nil
}

func transformDebtsToEmbed(debts *models.AllDebtsResponse) *discord.Embed {
	maxLength := len(slices.MaxFunc(debts.Debts, func(d1, d2 models.PlayerDebt) int {
		return len(d1.Name) - len(d2.Name)
	}).Name)
	debtString := strings.Builder{}
	debtString.WriteString("```")
	caser := cases.Title(language.English)
	for _, d := range debts.Debts {
		debtString.WriteString(fmt.Sprintf("%-*s %v\n", maxLength, caser.String(d.Name), d.Amount))
	}
	debtString.WriteString("```")

	version := os.Getenv("VERSION")
	return &discord.Embed{
		Title:       ":moneybag: 10k in die Gildenbank!",
		Type:        discord.NormalEmbed,
		Description: "[GitHub](https://github.com/torfstack/scurvy10k) | v" + version,
		Timestamp:   discord.NowTimestamp(),
		Color:       discord.Color(0xF1C40F),
		Footer: &discord.EmbedFooter{
			Text: "/10k <Spieler> <Betrag>",
			Icon: "https://true.torfstack.com/coin.png",
		},
		Fields: []discord.EmbedField{
			{
				Name:   "Spieler",
				Value:  debtString.String(),
				Inline: false,
			},
		},
	}
}

func updateDebtsMessage(s *state.State) {
	debts, err := getDebts()
	if err != nil {
		log.Error().Msgf("cannot get debts: %s", err)
		return
	}

	m, err := s.EditMessage(channelId, messageId, "", *transformDebtsToEmbed(debts))
	if err != nil {
		log.Error().Msgf("cannot edit message: %s", err)
		return
	}
	messageId = m.ID
}

func ephemeralMessage(content string) *api.InteractionResponseData {
	return &api.InteractionResponseData{
		Content: option.NewNullableString(content),
		Flags:   discord.EphemeralMessage,
	}
}
