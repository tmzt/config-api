--  Inspired by https://gist.github.com/phillip-haydon/54871b746201793990a18717af8d70dc#file-jsonb_merge-sql
--  TODO: Support arrays
CREATE OR REPLACE FUNCTION jsonb_merge(left JSONB, right JSONB) RETURNS JSONB AS $$

const mergeObjectInto = (dest, src) => {
    const isObject = (obj) => obj && typeof obj === 'object' && !Array.isArray(obj);

    for (const key in src) {
        if (src.hasOwnProperty(key)) {
            if (dest.hasOwnProperty(key) && isObject(dest[key]) && isObject(src[key])){
                mergeObjectInto(dest[key], src[key]);
            } else {
                dest[key] = src[key];
            }
        }
    }
    return dest;
};

const dest = Object.assign({}, left);

return mergeObjectInto(dest, right);

$$ LANGUAGE plv8;