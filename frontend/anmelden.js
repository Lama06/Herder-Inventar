export default {
    emits: ["erfolg"],
    data() {
        return {
            benutzername: "",
            passwort: "",
            passwortSichtbar: false,
        }
    },
    computed: {
        passwortInputType() {
            if (this.passwortSichtbar) {
                return "text"
            }
            return "password"
        },
        absendenKnopfAktiviert() {
            return this.benutzername !== "" && this.passwort !== ""
        }
    },
    methods: {
        async absenden() {
            let response;
            try {
                response = await fetch("/api/auth/login/", {
                    method: "POST",
                    body: JSON.stringify({
                        benutzername: this.benutzername,
                        passwort: this.passwort,
                    })
                })
            } catch {
                alert("Sever antwortet nicht")
                return
            }

            if (response.status === 401) {
                alert("Falsches Anmeldedaten")
                return
            }

            if (!response.ok) {
                alert("Serverfehler")
                return
            }

            let daten;
            try {
                daten = await response.json()
            } catch {
                alert("Server antwortet inkorrekt")
                return
            }

            sessionStorage.setItem("schlüssel", daten.schlüssel)
            this.$emit("erfolg")
        }
    },
    template: `
<h1>Anmelden</h1>
<label for="benutzername-eingabe">Benutzername:</label><input type="text" id="benutzername-eingabe" v-model="benutzername">
<br>
<label for="passwort-eingabe">Passwort: </label>
<input :type="passwortInputType" id="passwort-eingabe" v-model="passwort">
<button @click="passwortSichtbar = !passwortSichtbar">{{ passwortSichtbar ? "Passwort verstecken" : "Passwort anzeigen" }}</button>
<br>
<button :disabled="!absendenKnopfAktiviert" @click="absenden">Abenden</button>
`
}