// Copyright 2014 Unknwon
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

// Package i18n is a middleware that provides app Internationalization and Localization of Macaron.
package i18n

import (
	"fmt"
	"path"
	"strings"

	"github.com/Unknwon/com"
	"github.com/Unknwon/i18n"
	"github.com/Unknwon/macaron"
)

func Version() string {
	return "0.0.2"
}

// Initialized language type list.
func initLocales(opt Options) {
	for i, lang := range opt.Langs {
		fname := fmt.Sprintf(opt.Format, lang)
		// Append custom locale file.
		custom := []string{}
		customPath := path.Join(opt.CustomDirectory, fname)
		if com.IsFile(customPath) {
			custom = append(custom, customPath)
		}
		if err := i18n.SetMessageWithDesc(lang, opt.Names[i],
			path.Join(opt.Directory, fname), custom...); err != nil {
			panic(fmt.Errorf("fail to set message file(%s): %v", lang, err))
		}
	}
}

// A Locale describles the information of localization.
type Locale struct {
	i18n.Locale
}

// Language returns language current locale represents.
func (l Locale) Language() string {
	return l.Lang
}

// Options represents a struct for specifying configuration options for the i18n middleware.
type Options struct {
	// Suburl of path. Default is empty.
	SubURL string
	// Directory to load locale files. Default is "conf/locale"
	Directory string
	// Custom directory to overload locale files. Default is "custom/conf/locale"
	CustomDirectory string
	// Langauges that will be supported, order is meaningful.
	Langs []string
	// Human friendly names corresponding to Langs list.
	Names []string
	// Locale file naming style. Default is "locale_%s.ini".
	Format string
	// Name of language parameter name in URL. Default is "lang".
	Parameter string
	// Redirect when user uses get parameter to specify language.
	Redirect bool
	// Name that maps into template variable. Default is "i18n".
	TmplName string
}

func prepareOptions(options []Options) Options {
	var opt Options
	if len(options) > 0 {
		opt = options[0]
	}

	opt.SubURL = strings.TrimSuffix(opt.SubURL, "/")

	// Defaults
	if len(opt.Langs) == 0 {
		panic("no language is specified")
	} else if len(opt.Langs) != len(opt.Names) {
		panic("length of langs is not same as length of names")
	}

	if len(opt.Directory) == 0 {
		opt.Directory = "conf/locale"
	}
	if len(opt.CustomDirectory) == 0 {
		opt.CustomDirectory = "custom/conf/locale"
	}
	if len(opt.Format) == 0 {
		opt.Format = "locale_%s.ini"
	}
	if len(opt.Parameter) == 0 {
		opt.Parameter = "lang"
	}
	if len(opt.TmplName) == 0 {
		opt.TmplName = "i18n"
	}

	return opt
}

type LangType struct {
	Lang, Name string
}

// I18n is a middleware provides localization layer for your application.
// Paramenter langs must be in the form of "en-US", "zh-CN", etc.
// Otherwise it may not recognize browser input.
func I18n(options ...Options) macaron.Handler {
	opt := prepareOptions(options)
	initLocales(opt)
	return func(ctx *macaron.Context) {
		if ctx.Data == nil {
			panic("Render middleware is not registered")
		}

		isNeedRedir := false
		hasCookie := false

		// 1. Check URL arguments.
		lang := ctx.Query(opt.Parameter)

		// 2. Get language information from cookies.
		if len(lang) == 0 {
			lang = ctx.GetCookie("lang")
			hasCookie = true
		} else {
			isNeedRedir = true
		}

		// Check again in case someone modify by purpose.
		if !i18n.IsExist(lang) {
			lang = ""
			isNeedRedir = false
			hasCookie = false
		}

		// 3. Get language information from 'Accept-Language'.
		if len(lang) == 0 {
			al := ctx.Req.Header.Get("Accept-Language")
			if len(al) > 4 {
				al = al[:5] // Only compare first 5 letters.
				if i18n.IsExist(al) {
					lang = al
				}
			}
		}

		// 4. Default language is the first element in the list.
		if len(lang) == 0 {
			lang = i18n.GetLangByIndex(0)
			isNeedRedir = false
		}

		curLang := LangType{
			Lang: lang,
		}

		// Save language information in cookies.
		if !hasCookie {
			ctx.SetCookie("lang", curLang.Lang, 1<<31-1, "/"+strings.TrimPrefix(opt.SubURL, "/"))
		}

		restLangs := make([]LangType, 0, i18n.Count()-1)
		langs := i18n.ListLangs()
		names := i18n.ListLangDescs()
		for i, v := range langs {
			if lang != v {
				restLangs = append(restLangs, LangType{v, names[i]})
			} else {
				curLang.Name = names[i]
			}
		}

		// Set language properties.
		locale := Locale{i18n.Locale{lang}}
		ctx.Map(locale)
		ctx.Locale = locale
		ctx.Data[opt.TmplName] = locale
		ctx.Data["Tr"] = i18n.Tr
		ctx.Data["Lang"] = locale.Lang
		ctx.Data["AllLangs"] = append([]LangType{curLang}, restLangs...)
		ctx.Data["RestLangs"] = restLangs

		if opt.Redirect && isNeedRedir {
			ctx.Redirect(opt.SubURL + ctx.Req.RequestURI[:strings.Index(ctx.Req.RequestURI, "?")])
		}
	}
}
