// Code generated by templ - DO NOT EDIT.

// templ: version: v0.2.636
package views

//lint:file-ignore SA4006 This context is only used if a nested component is present.

import "github.com/a-h/templ"
import "context"
import "io"
import "bytes"

func About() templ.Component {
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
		templ_7745c5c3_Var2 := templ.ComponentFunc(func(ctx context.Context, templ_7745c5c3_W io.Writer) (templ_7745c5c3_Err error) {
			templ_7745c5c3_Buffer, templ_7745c5c3_IsBuffer := templ_7745c5c3_W.(*bytes.Buffer)
			if !templ_7745c5c3_IsBuffer {
				templ_7745c5c3_Buffer = templ.GetBuffer()
				defer templ.ReleaseBuffer(templ_7745c5c3_Buffer)
			}
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<div class=\"space-y-8\"><div><div><span class=\"text-3xl\">Definition</span></div><div><div><span class=\"text-2xl\">salt·y</span></div><div><span class=\"text-lg\">/ˈsôltē,ˈsältē/</span></div><div><a href=\"https://www.urbandictionary.com/define.php?term=salty\"><u>Being salty is when you are upset over something little.</u></a></div><div><span>Soojavu was so salty after reading a SooToday article.</span></div></div></div><div><div><span class=\"text-3xl\">What is this?</span></div><div><span>A website that ranks both the comments and users on various news sites. We started with </span> <a href=\"https://sootoday.com\">SooToday</a> <span>but have expanded to other sites in the same news network. The ranking of both comments and users is based on the number of likes and dislikes they've accumulated. Likes count for one point, dislikes count for two.</span></div></div><div><div><span class=\"text-3xl\">Your site is broken!</span></div><div><span>Probably. Let me know about it </span> <a href=\"https://github.com/salt-today/salttoday2/issues\"><u>here</u></a> <span>, or instead of being salty, try contributing?</span></div></div></div>")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			if !templ_7745c5c3_IsBuffer {
				_, templ_7745c5c3_Err = io.Copy(templ_7745c5c3_W, templ_7745c5c3_Buffer)
			}
			return templ_7745c5c3_Err
		})
		templ_7745c5c3_Err = Page(true).Render(templ.WithChildren(ctx, templ_7745c5c3_Var2), templ_7745c5c3_Buffer)
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		if !templ_7745c5c3_IsBuffer {
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteTo(templ_7745c5c3_W)
		}
		return templ_7745c5c3_Err
	})
}
