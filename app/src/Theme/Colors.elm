module Theme.Colors exposing (Palette, Theme(..), frappe, latte, macchiato, mocha)

import Element exposing (Color, rgb255)


type Theme
    = Latte
    | Frappe
    | Macchiato
    | Mocha


type alias Palette =
    { primary : Color
    , rosewater : Color
    , flamingo : Color
    , pink : Color
    , mauve : Color
    , red : Color
    , maroon : Color
    , peach : Color
    , yellow : Color
    , green : Color
    , teal : Color
    , sky : Color
    , sapphire : Color
    , blue : Color
    , lavender : Color
    , text : Color
    , subtext1 : Color
    , subtext0 : Color
    , overlay2 : Color
    , overlay1 : Color
    , overlay0 : Color
    , surface2 : Color
    , surface1 : Color
    , surface0 : Color
    , base : Color
    , mantle : Color
    , crust : Color
    }


latte : Palette
latte =
    { primary = rgb255 220 138 120
    , rosewater = rgb255 220 138 120
    , flamingo = rgb255 221 120 120
    , pink = rgb255 234 118 203
    , mauve = rgb255 136 57 239
    , red = rgb255 210 15 57
    , maroon = rgb255 230 69 83
    , peach = rgb255 254 100 11
    , yellow = rgb255 223 142 29
    , green = rgb255 64 160 43
    , teal = rgb255 23 146 153
    , sky = rgb255 4 165 229
    , sapphire = rgb255 32 159 181
    , blue = rgb255 30 102 245
    , lavender = rgb255 114 135 253
    , text = rgb255 76 79 105
    , subtext1 = rgb255 92 95 119
    , subtext0 = rgb255 108 111 133
    , overlay2 = rgb255 124 127 147
    , overlay1 = rgb255 140 143 161
    , overlay0 = rgb255 156 160 176
    , surface2 = rgb255 172 176 190
    , surface1 = rgb255 188 192 204
    , surface0 = rgb255 204 208 218
    , base = rgb255 239 241 245
    , mantle = rgb255 230 233 239
    , crust = rgb255 220 224 232
    }


frappe : Palette
frappe =
    { primary = rgb255 202 158 230
    , rosewater = rgb255 242 213 207
    , flamingo = rgb255 238 190 190
    , pink = rgb255 244 184 228
    , mauve = rgb255 202 158 230
    , red = rgb255 231 130 132
    , maroon = rgb255 234 153 156
    , peach = rgb255 239 159 118
    , yellow = rgb255 229 200 144
    , green = rgb255 166 209 137
    , teal = rgb255 129 200 190
    , sky = rgb255 153 209 219
    , sapphire = rgb255 133 193 220
    , blue = rgb255 140 170 238
    , lavender = rgb255 186 187 241
    , text = rgb255 198 208 245
    , subtext1 = rgb255 181 191 226
    , subtext0 = rgb255 165 173 206
    , overlay2 = rgb255 148 156 187
    , overlay1 = rgb255 131 139 167
    , overlay0 = rgb255 115 121 148
    , surface2 = rgb255 98 104 128
    , surface1 = rgb255 81 87 109
    , surface0 = rgb255 65 69 89
    , base = rgb255 48 52 70
    , mantle = rgb255 41 44 60
    , crust = rgb255 35 38 52
    }


macchiato : Palette
macchiato =
    { primary = rgb255 244 219 214
    , rosewater = rgb255 244 219 214
    , flamingo = rgb255 240 198 198
    , pink = rgb255 245 189 230
    , mauve = rgb255 198 160 246
    , red = rgb255 237 135 150
    , maroon = rgb255 238 153 160
    , peach = rgb255 245 169 127
    , yellow = rgb255 238 212 159
    , green = rgb255 166 218 149
    , teal = rgb255 139 213 202
    , sky = rgb255 145 215 227
    , sapphire = rgb255 125 196 228
    , blue = rgb255 138 173 244
    , lavender = rgb255 183 189 248
    , text = rgb255 202 211 245
    , subtext1 = rgb255 184 192 224
    , subtext0 = rgb255 165 173 203
    , overlay2 = rgb255 147 154 183
    , overlay1 = rgb255 128 135 162
    , overlay0 = rgb255 110 115 141
    , surface2 = rgb255 91 96 120
    , surface1 = rgb255 73 77 100
    , surface0 = rgb255 54 58 79
    , base = rgb255 36 39 58
    , mantle = rgb255 30 32 48
    , crust = rgb255 24 25 38
    }


mocha : Palette
mocha =
    { primary = rgb255 245 224 220
    , rosewater = rgb255 245 224 220
    , flamingo = rgb255 242 205 205
    , pink = rgb255 245 194 231
    , mauve = rgb255 203 166 247
    , red = rgb255 243 139 168
    , maroon = rgb255 235 160 172
    , peach = rgb255 250 179 135
    , yellow = rgb255 249 226 175
    , green = rgb255 166 227 161
    , teal = rgb255 148 226 213
    , sky = rgb255 137 220 235
    , sapphire = rgb255 116 199 236
    , blue = rgb255 137 180 250
    , lavender = rgb255 180 190 254
    , text = rgb255 205 214 244
    , subtext1 = rgb255 186 194 222
    , subtext0 = rgb255 166 173 200
    , overlay2 = rgb255 147 153 178
    , overlay1 = rgb255 127 132 156
    , overlay0 = rgb255 108 112 134
    , surface2 = rgb255 88 91 112
    , surface1 = rgb255 69 71 90
    , surface0 = rgb255 49 50 68
    , base = rgb255 30 30 46
    , mantle = rgb255 24 24 37
    , crust = rgb255 17 17 27
    }

