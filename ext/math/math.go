package math

import (
	"math"
	"github.com/bobappleyard/ts"
)

func init() {
	ts.RegisterExtension("math", pkg)
}

func pkg(it *ts.Interpreter) map[string] *ts.Object {
	toFloat := it.Accessor("toFloat")

	flt := func(x *ts.Object) float64 {
		return x.Call(toFloat).ToFloat()
	}
	
	wrap1 := func(f func(a float64) float64) *ts.Object {
		return ts.Wrap(func(o, a *ts.Object) *ts.Object {
			return ts.Wrap(f(flt(a)))
		})
	}

	wrap2 := func(f func(a, b float64) float64) *ts.Object {
		return ts.Wrap(func(o, a, b *ts.Object) *ts.Object {
			return ts.Wrap(f(flt(a), flt(b)))
		})
	}

	return map[string] *ts.Object {
		"E": ts.Wrap(math.E),
		"PI": ts.Wrap(math.Pi),
		"PHI": ts.Wrap(math.Phi),
		"SQRT2": ts.Wrap(math.Sqrt2),
		"SQRTE": ts.Wrap(math.SqrtE),
		"SQRTPI": ts.Wrap(math.SqrtPi),
		"SQRTPHI": ts.Wrap(math.SqrtPhi),
		"LN2": ts.Wrap(math.Ln2),
		"LOG2E": ts.Wrap(math.Log2E),
		"LN10": ts.Wrap(math.Ln10),
		"LOG10E": ts.Wrap(math.Log10E),
		"abs": wrap1(math.Abs),
		"acos": wrap1(math.Acos),
		"acosh": wrap1(math.Acosh),
		"asin": wrap1(math.Asin),
		"asinh": wrap1(math.Asinh),
		"atan": wrap1(math.Atan),
		"atanh": wrap1(math.Atanh),
		"cbrt": wrap1(math.Cbrt),
		"ceil": wrap1(math.Ceil),
		"cos": wrap1(math.Cos),
		"cosh": wrap1(math.Cosh),
		"erf": wrap1(math.Erf),
		"erfc": wrap1(math.Erfc),
		"exp": wrap1(math.Exp),
		"exp2": wrap1(math.Exp2),
		"expm1": wrap1(math.Expm1),
		"floor": wrap1(math.Floor),
		"gamma": wrap1(math.Gamma),
		"j0": wrap1(math.J0),
		"j1": wrap1(math.J1),
		"log": wrap1(math.Log),
		"log10": wrap1(math.Log10),
		"log1p": wrap1(math.Log1p),
		"log2": wrap1(math.Log2),
		"logb": wrap1(math.Logb),
		"sin": wrap1(math.Sin),
		"sinh": wrap1(math.Sinh),
		"sqrt": wrap1(math.Sqrt),
		"tan": wrap1(math.Tan),
		"tanh": wrap1(math.Tanh),
		"trunc": wrap1(math.Trunc),
		"y0": wrap1(math.Y0),
		"y1": wrap1(math.Y1),
		"atan2": wrap2(math.Atan2),
		"copysign": wrap2(math.Copysign),
		"dim": wrap2(math.Dim),
		"hypot": wrap2(math.Hypot),
		"max": wrap2(math.Max),
		"min": wrap2(math.Min),
		"mod": wrap2(math.Mod),
		"nextafter": wrap2(math.Nextafter),
		"pow": wrap2(math.Pow),
		"remainder": wrap2(math.Remainder),
		"ilogb": ts.Wrap(func(o, x *ts.Object) *ts.Object {
			return ts.Wrap(math.Ilogb(flt(x)))
		}),
		"inf": ts.Wrap(func(o, sign *ts.Object) *ts.Object {
			return ts.Wrap(math.Inf(sign.ToInt()))
		}),
		"isInf": ts.Wrap(func(o, x, sign *ts.Object) *ts.Object {
			return ts.Wrap(math.IsInf(flt(x), sign.ToInt()))
		}),
		"isNaN": ts.Wrap(func(o, x *ts.Object) *ts.Object {
			return ts.Wrap(math.IsNaN(flt(x)))
		}),
		"NaN": ts.Wrap(math.NaN()),
		"pow10": ts.Wrap(func(o, e *ts.Object) *ts.Object {
			return ts.Wrap(math.Pow10(e.ToInt()))
		}),
		"jn": ts.Wrap(func(o, n, x *ts.Object) *ts.Object {
			return ts.Wrap(math.Jn(n.ToInt(), flt(x)))
		}),
		"yn": ts.Wrap(func(o, n, x *ts.Object) *ts.Object {
			return ts.Wrap(math.Yn(n.ToInt(), flt(x)))
		}),
	}

}
