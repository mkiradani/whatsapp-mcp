package main

import "testing"

// Guardián del bug "los chats @lid se guardan con el número como nombre".
//
// Caso canónico observado en producción (2026-07-16): un chat cuyo contacto se
// llama "Ada Lovelace" se guardaba en messages.db como "15550001111" (su LID),
// así que buscar el contacto por nombre no encontraba nada. 163 de ~200 chats
// afectados. WhatsApp guarda DOS filas del mismo contacto en whatsmeow_contacts:
//
//	their_jid                      full_name       push_name
//	<telefono>@s.whatsapp.net      Ada Lovelace    Ada Lovelace
//	<lid>@lid                      (vacío)         Ada Lovelace
//
// El chat vive bajo el @lid, cuya fila tiene full_name VACÍO. GetChatName sólo
// miraba FullName, así que caía al fallback y guardaba el identificador. El
// nombre estaba en push_name y nadie lo leía.
//
// (Datos de ejemplo ficticios: el caso real vive en el ledger privado.)

func TestPickContactName_LIDContactSoloTienePushName(t *testing.T) {
	// La fila @lid: sin full_name, con push_name.
	got := pickContactName(contactNames{FullName: "", PushName: "Ada Lovelace"})
	if got != "Ada Lovelace" {
		t.Errorf("un contacto @lid con sólo push_name debe resolver a su nombre real:\n  quiero: %q\n  tengo:  %q", "Ada Lovelace", got)
	}
}

func TestPickContactName_FullNameManda(t *testing.T) {
	// El nombre de la libreta gana al que el contacto se pone a sí mismo.
	got := pickContactName(contactNames{FullName: "Ada Lovelace", PushName: "ada🏍️"})
	if got != "Ada Lovelace" {
		t.Errorf("full_name debe tener prioridad:\n  quiero: %q\n  tengo:  %q", "Ada Lovelace", got)
	}
}

func TestPickContactName_SinNingunNombre(t *testing.T) {
	// Sin ningún dato utilizable devuelve "" para que el llamante haga fallback.
	if got := pickContactName(contactNames{}); got != "" {
		t.Errorf("un contacto sin nombres debe devolver cadena vacía, tengo: %q", got)
	}
}

func TestPickContactName_BusinessYFirstName(t *testing.T) {
	if got := pickContactName(contactNames{BusinessName: "Hotel Example"}); got != "Hotel Example" {
		t.Errorf("business_name debe valer como nombre, tengo: %q", got)
	}
	if got := pickContactName(contactNames{FirstName: "Ada"}); got != "Ada" {
		t.Errorf("first_name debe valer como último recurso, tengo: %q", got)
	}
}

// El segundo fallo: una vez guardado el número como nombre, GetChatName lo
// reutilizaba para siempre. Sin esto, arreglar pickContactName no cura los
// chats que ya estaban envenenados.

func TestIsPlaceholderName_ElNumeroGuardadoNoEsUnNombre(t *testing.T) {
	if !isPlaceholderName("15550001111", "15550001111") {
		t.Error("el LID guardado como nombre es un placeholder: debe volver a resolverse")
	}
	if !isPlaceholderName("34600000000", "34600000000") {
		t.Error("un teléfono guardado como nombre es un placeholder: debe volver a resolverse")
	}
	if !isPlaceholderName("", "34600000000") {
		t.Error("un nombre vacío es un placeholder")
	}
}

func TestIsPlaceholderName_LosNombresBuenosSobreviven(t *testing.T) {
	// Contrapeso: el arreglo NO debe re-resolver ni pisar los nombres ya correctos.
	buenos := []struct{ name, jidUser string }{
		{"Ada", "34600000000"},
		{"Ada Lovelace", "15550001111"},
		{"Mamá", "34600000001"},
		{"Grupo ESPAÑA🇪🇦🇪🇦🇪🇦", "120363000000000000"},
		{"Grace Hopper", "34600000002"},
	}
	for _, b := range buenos {
		if isPlaceholderName(b.name, b.jidUser) {
			t.Errorf("%q es un nombre real y no debe tratarse como placeholder", b.name)
		}
	}
}
