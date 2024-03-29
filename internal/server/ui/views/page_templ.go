// Code generated by templ - DO NOT EDIT.

// templ: version: v0.2.636
package views

//lint:file-ignore SA4006 This context is only used if a nested component is present.

import "github.com/a-h/templ"
import "context"
import "io"
import "bytes"

func Page(nav bool) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, templ_7745c5c3_W io.Writer) (templ_7745c5c3_Err error) {
		templ_7745c5c3_Buffer, templ_7745c5c3_IsBuffer := templ_7745c5c3_W.(*bytes.Buffer)
		if !templ_7745c5c3_IsBuffer {
			templ_7745c5c3_Buffer = templ.GetBuffer()
			defer templ.ReleaseBuffer(templ_7745c5c3_Buffer)
		}
		ctx = templ.InitializeContext(ctx)
		templ_7745c5c3_Var1 := templ.GetChildren(ctx)
		if templ_7745c5c3_Var1 == nil {
			templ_7745c5c3_Var1 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<!doctype html><html lang=\"en\"><head><title>SaltToday 2</title><link rel=\"icon\" type=\"image/x-icon\" href=\"/public/images/SaltTodayLogoRedBlue.psd\"><link rel=\"icon\" href=\"/public/output.css\"><meta charset=\"UTF-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\"><link rel=\"stylesheet\" href=\"/public/output.css\"><script src=\"https://unpkg.com/htmx.org@1.9.10\" defer></script></head><body class=\"font-sans leading-tight bg-slate-900 text-white bg-[url(&#39;/public/images/salt-falling.png&#39;)] bg-center bg-repeat-y min-h-screen\">")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		if nav {
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<nav class=\"border-black\"><div class=\"bg-slate-950 max-w flex flex-wrap items-center justify-between mx-auto p-4\"><a href=\"/\" class=\"flex items-center space-x-3 rtl:space-x-reverse\"><img src=\"/public/images/SaltTodayLogoRedBlue.psd\" class=\"h-8\" alt=\"SaltToday Logo\"> <span class=\"self-center text-2xl font-semibold whitespace-norap text-white\">SaltToday</span></a><div class=\"hidden w-full md:block md:w-auto\" id=\"navbar-default\"><ul class=\"font-medium flex flex-col p-4 md:p-0 mt-4 border border-slate-100 rounded-lg bg-slate-950 md:flex-row md:space-x-8 rtl:space-x-reverse md:mt-0 md:border-0\"><li><a href=\"/\" class=\"block py-2 px-3 text-white rounded md:bg-transparent md:p-0\" aria-current=\"page\">Home</a></li><li><a href=\"/users\" class=\"block py-2 px-3 text-white rounded md:bg-transparent md:p-0\">Users</a></li><li><a href=\"/about\" class=\"block py-2 px-3 text-white rounded md:bg-transparent md:p-0\">About</a></li></ul></div></div></nav>")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<div class=\"max-w-screen-xl p-9 min-h-80 items-center justify-between mx-auto bg-slate-900 bg-transparent\">")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		templ_7745c5c3_Err = templ_7745c5c3_Var1.Render(ctx, templ_7745c5c3_Buffer)
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("</div><div class=\"flex flex-col justify-end\"><img class=\"\" src=\"/public/images/footer-pile.png\" alt=\"Pile of salt\"></div></body></html>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		if !templ_7745c5c3_IsBuffer {
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteTo(templ_7745c5c3_W)
		}
		return templ_7745c5c3_Err
	})
}