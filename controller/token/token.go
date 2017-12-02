package token

import (
	"fmt"
	"github.com/pdbogen/mapbot/common/colors"
	"github.com/pdbogen/mapbot/common/conv"
	"github.com/pdbogen/mapbot/common/db"
	mbLog "github.com/pdbogen/mapbot/common/log"
	"github.com/pdbogen/mapbot/controller/cmdproc"
	"github.com/pdbogen/mapbot/hub"
	"github.com/pdbogen/mapbot/model/tabula"
	"github.com/pdbogen/mapbot/model/types"
	"image/color"
	"reflect"
)

var log = mbLog.Log

func Register(h *hub.Hub) {
	h.Subscribe("user:token", processor.Route)
}

var processor *cmdproc.CommandProcessor

func init() {
	processor = &cmdproc.CommandProcessor{
		Command: "token",
		Commands: map[string]cmdproc.Subcommand{
			"add":    cmdproc.Subcommand{"<name> <X> <Y> [<name2> <x2> <y2> ... <nameN> <xN> <yN>]", "add a token(s) (or change its location) to the currently selected map (see `map select`). Token names should be emoji! (Or very short words).", cmdAdd},
			"move":   cmdproc.Subcommand{"<name> <X> <Y>", "synonym for add", cmdAdd},
			"color":  cmdproc.Subcommand{"<name> <color>", "sets the color for the given token, which can be a common name; the world 'clear'; a 6-digit hex code specifying red, green, and blue (optionally with two more digits specifying Alpha); https://en.wikipedia.org/wiki/List_of_Crayola_crayon_colors has a great list of colors.", cmdColor},
			"list":   cmdproc.Subcommand{"", "list tokens on the active map", cmdList},
			"clear":  cmdproc.Subcommand{"", "clear tokens from the field", cmdClear},
			"remove": cmdproc.Subcommand{"<name>", "removes the named token from the active map", cmdRemove},
			"swap":   cmdproc.Subcommand{"<old> <new>", "replace an old token with a new token, retaining other settings (location/color)", cmdSwap},
		},
	}
}

func cmdSwap(h *hub.Hub, c *hub.Command) {
	args, ok := c.Payload.([]string)
	if !ok {
		h.Error(c, "unexpected payload")
		log.Errorf("expected []string payload, but received %s", reflect.TypeOf(c.Payload))
		return
	}

	if len(args) != 2 {
		h.Error(c, "usage: token swap "+processor.Commands["swap"].Args)
		return
	}

	tabId := c.Context.GetActiveTabulaId()
	if tabId == nil {
		h.Error(c, "no active map in this channel, use `map select <name>` first")
		return
	}

	tab, err := tabula.Get(db.Instance, *tabId)
	if err != nil {
		h.Error(c, "an error occured loading the active map for this channel")
		log.Errorf("error loading tabula %d: %s", *tabId, err)
		return
	}

	name := args[0]

	if tab.Tokens == nil || tab.Tokens[c.Context.Id()] == nil {
		h.Error(c, fmt.Sprintf("no token %s is on the active map; try `token list`", name))
		return
	}

	token, tokenOk := tab.Tokens[c.Context.Id()][name]
	if !tokenOk {
		h.Error(c, fmt.Sprintf("no token %s is on the active map; try `token list`", name))
		return
	}

	tab.Tokens[c.Context.Id()][args[1]] = token
	delete(tab.Tokens[c.Context.Id()], name)

	if err := tab.Save(db.Instance); err != nil {
		h.Error(c, "an error occured saving the active map for this channel")
		log.Errorf("error saving tabula %d: %s", tab.Id, err)
	}

	h.Publish(c.WithType(hub.CommandType(c.From)).WithPayload(tab))
}

func cmdColor(h *hub.Hub, c *hub.Command) {
	args, ok := c.Payload.([]string)
	if !ok {
		h.Error(c, "unexpected payload")
		log.Errorf("expected []string payload, but received %s", reflect.TypeOf(c.Payload))
		return
	}

	if len(args) != 2 {
		h.Error(c, "usage: token color "+processor.Commands["color"].Args)
		return
	}

	tabId := c.Context.GetActiveTabulaId()
	if tabId == nil {
		h.Error(c, "no active map in this channel, use `map select <name>` first")
		return
	}

	tab, err := tabula.Get(db.Instance, *tabId)
	if err != nil {
		h.Error(c, "an error occured loading the active map for this channel")
		log.Errorf("error loading tabula %d: %s", *tabId, err)
		return
	}

	name := args[0]

	if tab.Tokens == nil || tab.Tokens[c.Context.Id()] == nil {
		h.Error(c, fmt.Sprintf("no token %s is on the active map; try `token list`", name))
		return
	}

	token, tokenOk := tab.Tokens[c.Context.Id()][name]
	if !tokenOk {
		h.Error(c, fmt.Sprintf("no token %s is on the active map; try `token list`", name))
		return
	}

	newColor, err := colors.ToColor(args[1])
	if err != nil {
		h.Error(c, err.Error())
		return
	}

	tab.Tokens[c.Context.Id()][name] = token.WithColor(newColor)

	if err := tab.Save(db.Instance); err != nil {
		h.Error(c, "an error occured saving the active map for this channel")
		log.Errorf("error saving tabula %d: %s", tab.Id, err)
	}

	h.Publish(c.WithType(hub.CommandType(c.From)).WithPayload(tab))
}

func cmdList(h *hub.Hub, c *hub.Command) {
	tabId := c.Context.GetActiveTabulaId()
	if tabId == nil {
		h.Error(c, "no active map in this channel, use `map select <name>` first")
		return
	}

	tab, err := tabula.Get(db.Instance, *tabId)
	if err != nil {
		h.Error(c, "an error occured loading the active map for this channel")
		log.Errorf("error loading tabula %d: %s", *tabId, err)
		return
	}

	ctxId := c.Context.Id()
	if tab.Tokens == nil || tab.Tokens[ctxId] == nil || len(tab.Tokens[ctxId]) == 0 {
		h.Reply(c, "There are no tokens on the active map.")
		return
	}

	rep := fmt.Sprintf("There are %d tokens on the active map:", len(tab.Tokens[ctxId]))
	for name, token := range tab.Tokens[ctxId] {
		if name[0] == ':' && name[len(name)-1] == ':' {
			name = name + "(`" + name + "`)"
		}
		r, g, b, a := token.TokenColor.RGBA()
		rep += fmt.Sprintf("\n- %s at (%d,%d), color (%d,%d,%d,%d)", name, token.Coordinate.X, token.Coordinate.Y, r, g, b, a)
	}
	h.Reply(c, rep)
	return
}

func cmdRemove(h *hub.Hub, c *hub.Command) {
	args, ok := c.Payload.([]string)
	if !ok {
		h.Error(c, "unexpected payload")
		log.Errorf("expected []string payload, but received %s", reflect.TypeOf(c.Payload))
		return
	}

	if len(args) == 0 {
		h.Error(c, "`token remove` expects a list of tokens to clear. usage: `token clear "+processor.Commands["remove"].Args+"`")
		return
	}

	tabId := c.Context.GetActiveTabulaId()
	if tabId == nil {
		h.Error(c, "no active map in this channel, use `map select <name>` first")
		return
	}

	tab, err := tabula.Get(db.Instance, *tabId)
	if err != nil {
		h.Error(c, "an error occured loading the active map for this channel")
		log.Errorf("error loading tabula %d: %s", *tabId, err)
		return
	}

	for _, token := range args {
		log.Debugf("removing token %s", token)
		delete(tab.Tokens[c.Context.Id()], token)
	}

	if err := tab.Save(db.Instance); err != nil {
		h.Error(c, "an error occured saving the active map for this channel")
		log.Errorf("error saving tabula %d: %s", tab.Id, err)
	}

	h.Publish(c.WithType(hub.CommandType(c.From)).WithPayload(tab))
}

func cmdClear(h *hub.Hub, c *hub.Command) {
	args, ok := c.Payload.([]string)
	if !ok {
		h.Error(c, "unexpected payload")
		log.Errorf("expected []string payload, but received %s", reflect.TypeOf(c.Payload))
		return
	}

	if len(args) > 0 {
		h.Error(c, "`token clear` expects no arguments; usage: token clear")
		return
	}

	tabId := c.Context.GetActiveTabulaId()
	if tabId == nil {
		h.Error(c, "no active map in this channel, use `map select <name>` first")
		return
	}

	tab, err := tabula.Get(db.Instance, *tabId)
	if err != nil {
		h.Error(c, "an error occured loading the active map for this channel")
		log.Errorf("error loading tabula %d: %s", *tabId, err)
		return
	}

	tab.Tokens[c.Context.Id()] = map[string]tabula.Token{}

	if err := tab.Save(db.Instance); err != nil {
		h.Error(c, "an error occured saving the active map for this channel")
		log.Errorf("error saving tabula %d: %s", tab.Id, err)
	}

	h.Publish(c.WithType(hub.CommandType(c.From)).WithPayload(tab))
}

func cmdAdd(h *hub.Hub, c *hub.Command) {
	args, ok := c.Payload.([]string)
	if !ok {
		h.Error(c, "unexpected payload")
		log.Errorf("expected []string payload, but received %s", reflect.TypeOf(c.Payload))
		return
	}

	if len(args) < 3 || len(args)%3 != 0 {
		h.Error(c, "`token add` expects 3 arguments per token; usage: token add "+processor.Commands["add"].Args)
		return
	}

	tabId := c.Context.GetActiveTabulaId()
	if tabId == nil {
		h.Error(c, "no active map in this channel, use `map select <name>` first")
		return
	}

	tab, err := tabula.Get(db.Instance, *tabId)
	if err != nil {
		h.Error(c, "an error occured loading the active map for this channel")
		log.Errorf("error loading tabula %d: %s", *tabId, err)
		return
	}

	for tokN := 0; tokN < len(args); tokN += 3 {
		name := args[tokN]

		coord, err := conv.CoordsToPoint(args[tokN+1], args[tokN+2])
		if err != nil {
			h.Error(c, fmt.Sprintf("Invalid coordinates: %s", err))
			return
		}

		if tab.Tokens == nil {
			tab.Tokens = map[types.ContextId]map[string]tabula.Token{}
		}

		if tab.Tokens[c.Context.Id()] == nil {
			tab.Tokens[c.Context.Id()] = map[string]tabula.Token{}
		}

		if tok, ok := tab.Tokens[c.Context.Id()][name]; !ok {
			tab.Tokens[c.Context.Id()][name] = tabula.Token{coord, color.RGBA{0, 0, 0, 0}}
		} else {
			tab.Tokens[c.Context.Id()][name] = tok.WithCoords(coord)
		}
	}

	if err := tab.Save(db.Instance); err != nil {
		h.Error(c, "an error occured saving the active map for this channel")
		log.Errorf("error saving tabula %d: %s", tab.Id, err)
	}

	h.Publish(c.WithType(hub.CommandType(c.From)).WithPayload(tab))
}
